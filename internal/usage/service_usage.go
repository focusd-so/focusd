package usage

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
	"gorm.io/gorm"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/internal/identity"
)

// IdleChanged is called when the idle state of the user changes (e.g. user starts or stops using the computer)
func (s *Service) IdleChanged(ctx context.Context, isIdle bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if isIdle {
		// close only non-idle application usage (leave any existing idle usage open)
		var currentUsage ApplicationUsage
		err := s.db.Preload("Application").
			Joins("LEFT JOIN application ON application.id = application_usage.application_id").
			Where("application_usage.ended_at IS NULL AND (application.name IS NULL OR application.name != ?)", IdleApplicationName).
			Limit(1).Order("application_usage.started_at DESC").
			First(&currentUsage).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to find current application usage: %w", err)
		}
		if currentUsage.ID > 0 {
			if err := s.closeApplicationUsage(&currentUsage); err != nil {
				return fmt.Errorf("failed to close current application usage: %w", err)
			}
		}

		// check if there is already an open idle usage
		var existingIdleUsage ApplicationUsage
		err = s.db.Joins("Application").
			Where("application.name = ? AND application_usage.ended_at IS NULL", IdleApplicationName).
			Limit(1).Order("application_usage.started_at DESC").
			First(&existingIdleUsage).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to find current idle usage: %w", err)
		}

		// if there is an open idle usage, let it continue
		if existingIdleUsage.ID > 0 {
			return nil
		}

		// get or create the Idle application
		var idleApp Application
		if err := s.db.Where("name = ?", IdleApplicationName).First(&idleApp).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				idleApp = Application{Name: IdleApplicationName}
				if err := s.db.Create(&idleApp).Error; err != nil {
					return fmt.Errorf("failed to create idle application: %w", err)
				}
			} else {
				return fmt.Errorf("failed to find idle application: %w", err)
			}
		}

		// create a new idle usage
		idleUsage := &ApplicationUsage{
			ApplicationID:   idleApp.ID,
			StartedAt:       time.Now().Unix(),
			Classification:  ClassificationNone,
			TerminationMode: TerminationModeNone,
			WindowTitle:     IdleApplicationName,
			ExecutablePath:  "idle",
		}
		if err := s.db.Create(idleUsage).Error; err != nil {
			return fmt.Errorf("failed to create idle usage: %w", err)
		}

	} else {
		// close the current idle usage
		if err := s.closeCurrentIdleUsage(); err != nil {
			return fmt.Errorf("failed to close current idle usage: %w", err)
		}
	}

	return nil
}

// TitleChanged is called when the title of the current application changes,
// whether it's a new application or the same application title has changed
func (s *Service) TitleChanged(ctx context.Context, executablePath, windowTitle, appName, icon string, bundleID, url *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// check if the current application is the same
	currentApplicationUsage, err := s.getCurrentApplicationUsage()
	if err != nil {
		return fmt.Errorf("failed to find current application usage: %w", err)
	}

	// if the current application is the same, let it continue
	if currentApplicationUsage != nil && currentApplicationUsage.Same(windowTitle, appName, bundleID, url) {
		return nil
	}

	// if new application usage is detected close the current application usage before creating a new one
	if err := s.closeApplicationUsage(currentApplicationUsage); err != nil {
		return fmt.Errorf("failed to close current application usage: %w", err)
	}

	application, err := s.getOrCreateApplication(ctx, appName, icon, bundleID, url)
	if err != nil {
		return fmt.Errorf("failed to get or create application: %w", err)
	}

	applicationUsage := ApplicationUsage{
		ExecutablePath: executablePath,
		WindowTitle:    windowTitle,
		BrowserURL:     url,
		StartedAt:      time.Now().Unix(),

		Application: application,
	}

	if s.UsageUpdates != nil {
		s.UsageUpdates <- &applicationUsage
	}

	// save the application usage
	if err := s.db.Save(&applicationUsage).Error; err != nil {
		return fmt.Errorf("failed to save application usage: %w", err)
	}

	classification, err := s.classifyApplicationUsage(ctx, &applicationUsage)
	if err != nil {
		errMsg := err.Error()
		applicationUsage.ClassificationError = &errMsg
		applicationUsage.Classification = ClassificationNone
	} else if classification != nil {
		applicationUsage.Classification = classification.Classification
		applicationUsage.ClassificationSource = &classification.ClassificationSource
		applicationUsage.ClassificationReasoning = &classification.Reasoning
		applicationUsage.ClassificationConfidence = &classification.ConfidenceScore

		if classification.DetectedProject != "" {
			applicationUsage.DetectedProject = &classification.DetectedProject
		}
		if classification.DetectedCommunicationChannel != "" {
			applicationUsage.DetectedCommunicationChannel = &classification.DetectedCommunicationChannel
		}
		applicationUsage.Tags = make([]ApplicationUsageTags, len(classification.Tags))
		for i, tag := range classification.Tags {
			applicationUsage.Tags[i] = ApplicationUsageTags{
				Tag: tag,
			}
		}

		if classification.SandboxContext != "" {
			applicationUsage.SandboxContext = &classification.SandboxContext
		}
		if classification.SandboxResponse != nil {
			applicationUsage.SandboxResponse = classification.SandboxResponse
		}
		if classification.SandboxLogs != "" {
			applicationUsage.SandboxLogs = &classification.SandboxLogs
		}
	}

	// calculate termination mode.
	terminationMode, err := s.CalculateTerminationMode(ctx, &applicationUsage)
	if err != nil {
		termErr := err.Error()
		applicationUsage.TerminationMode = TerminationModeNone
		applicationUsage.TerminationError = &termErr
	}

	applicationUsage.TerminationMode = terminationMode.Mode

	if terminationMode.Reasoning != "" {
		applicationUsage.TerminationReasoning = &terminationMode.Reasoning
	}
	if terminationMode.Source != "" {
		applicationUsage.TerminationSource = &terminationMode.Source
	}

	if err := s.db.Save(&applicationUsage).Error; err != nil {
		return fmt.Errorf("failed to save application usage: %w", err)
	}

	if applicationUsage.TerminationMode == TerminationModeBlock {
		termReason := ""
		if applicationUsage.ClassificationReasoning != nil {
			termReason = *applicationUsage.ClassificationReasoning
		}

		tags := ApplicationTagsSlice(applicationUsage.Tags).Tags()

		s.appBlocker(applicationUsage.Application.Name, applicationUsage.WindowTitle, termReason, tags, applicationUsage.BrowserURL)
	}

	if s.UsageUpdates != nil {
		s.UsageUpdates <- &applicationUsage
	}

	return nil
}

func (s *Service) classifyApplicationUsage(ctx context.Context, applicationUsage *ApplicationUsage) (*ClassificationResponse, error) {
	// Do sandbox classification first, eg user defined custom rules
	customRulesResp, err := s.ClassifyCustomRules(ctx, applicationUsage.Application.Name, applicationUsage.BrowserURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to classify application usage with custom rules: %w", err)
	}

	tier := identity.GetAccountTier()
	isPaid := tier != apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE

	if customRulesResp != nil && isPaid {
		return customRulesResp, nil
	}

	// Do obviously classification next, eg social media, news, shopping, etc.
	classification, err := s.classifyObviously(ctx, applicationUsage.Application.Name, applicationUsage.BrowserURL)
	if err != nil {
		return nil, fmt.Errorf("failed to classify application usage with obviously: %w", err)
	}

	if classification != nil {
		if customRulesResp != nil {
			classification.SandboxContext = customRulesResp.SandboxContext
			classification.SandboxResponse = customRulesResp.SandboxResponse
			classification.SandboxLogs = customRulesResp.SandboxLogs
		}
		return classification, nil
	}

	slog.Info("classifying application usage with LLM")
	resp, err := s.ClassifyWithLLM(ctx, applicationUsage.Application.Name, applicationUsage.WindowTitle, applicationUsage.BrowserURL)
	if err != nil {
		return nil, fmt.Errorf("failed to classify application usage with LLM: %w", err)
	}

	if customRulesResp != nil {
		resp.SandboxContext = customRulesResp.SandboxContext
		resp.SandboxResponse = customRulesResp.SandboxResponse
		resp.SandboxLogs = customRulesResp.SandboxLogs
	}

	slog.Info("classified application usage with LLM", "response", resp)

	return resp, nil
}

// getOrCreateApplication retrieves an existing application from the database or creates a new one.
//
// This function handles two distinct types of applications:
//
// 1. Web Applications (when rawURL is provided):
//   - Parses the URL to extract the hostname (e.g., "www.example.com")
//   - Looks up the application by hostname in the database
//   - If not found, creates a new Application record with the hostname
//   - Extracts the effective domain (TLD+1) using public suffix list
//     (e.g., "example.com" from "sub.example.com")
//   - Fetches the favicon from Google's favicon service if icon is not already set
//   - Saves and returns the application
//
// 2. Native Applications (when rawURL is nil):
//   - Looks up the application by executable path in the database
//   - If not found, creates a new Application record with the executable path
//   - Uses the provided icon directly (typically from the OS)
//   - Saves and returns the application
//
// Parameters:
//   - ctx: Context for cancellation and timeout control (used for favicon fetching)
//   - name: Display name of the application (e.g., "Safari", "Google Chrome")
//   - icon: Base64-encoded icon data for native apps (ignored for web apps)
//   - bundleID: Optional macOS bundle identifier (e.g., "com.apple.Safari")
//   - rawURL: Optional URL for web applications; if nil, treated as native app
//
// Returns:
//   - Application: The found or newly created application record
//   - error: Any error encountered during database operations or favicon fetching
func (s *Service) getOrCreateApplication(ctx context.Context, name, icon string, bundleID, rawURL *string) (Application, error) {
	// Handle web applications (browser tabs with URLs)

	rawURLValue := fromPtr(rawURL)

	if rawURLValue != "" {
		// Extract hostname from the URL (e.g., "www.google.com" from "https://www.google.com/search?q=...")
		u, err := url.Parse(rawURLValue)
		if err != nil {
			slog.Warn("failed to parse URL", "error", err)
		}

		hostname := u.Hostname()

		// Attempt to find an existing application record by hostname.
		// Web apps are uniquely identified by hostname, so all tabs from the same
		// site (e.g., multiple Google tabs) share the same Application record.
		var (
			application Application
			query       = s.db.Where("(hostname = ? OR hostname IS NULL) AND name = ?", hostname, name)
		)

		if err := query.First(&application).Error; err != nil {
			slog.Warn("failed to find application by hostname", "error", err)
		}

		// If no existing application found, create a new one with the provided metadata
		if application.ID == 0 {
			application = Application{Name: name, BundleID: bundleID}
		}

		if hostname != "" {
			application.Hostname = &hostname

			// Extract the effective domain (TLD+1) if not already set.
			// This normalizes subdomains: "mail.google.com" and "docs.google.com" both become "google.com".
			// Useful for grouping related sites and aggregating usage statistics.
			domain, _ := publicsuffix.EffectiveTLDPlusOne(hostname)
			if domain != "" {
				application.Domain = &domain
			}
		}

		// Fetch favicon if icon is empty or very small (old 16x16 ICO format, typically <500 chars base64)
		if application.Icon == nil {
			appIcon, err := fetchFavicon(ctx, fmt.Sprintf("https://%s", hostname))
			if err != nil {
				slog.Warn("failed to fetch app icon", "error", err)
			}

			if appIcon != "" {
				application.Icon = &appIcon
			}
		}

		if err := s.db.Save(&application).Error; err != nil {
			return Application{}, fmt.Errorf("failed to create application: %w", err)
		}

		return application, nil
	}

	// Handle native applications (non-browser apps identified by executable path)
	// Native apps are uniquely identified by their executable path on disk.
	var application Application
	if err := s.db.Where("name = ?", name).First(&application).Error; err != nil {
		slog.Warn("failed to find application by executable path", "error", err)
	}

	// If no existing application found, create a new one with the provided metadata.
	// For native apps, the icon is provided by the OS (e.g., from the app bundle).
	if application.ID == 0 {
		application = Application{
			Name:     name,
			BundleID: bundleID,
		}
	}

	slog.Info("application icon", "icon", application.Icon)

	if application.Icon == nil && icon != "" {
		// Update the icon for existing apps that don't have one yet
		application.Icon = &icon
	}

	// Persist the application (creates new or updates existing)
	if err := s.db.Save(&application).Error; err != nil {
		return Application{}, fmt.Errorf("failed to create application: %w", err)
	}

	return application, nil
}

func (s *Service) getCurrentApplicationUsage() (*ApplicationUsage, error) {
	var application ApplicationUsage

	if err := s.db.Preload("Application").Where("ended_at IS NULL").Limit(1).Order("started_at DESC").First(&application).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to find current application usage: %w", err)
		}
	}

	if application.ID == 0 {
		return nil, nil
	}

	return &application, nil
}

func (s *Service) closeApplicationUsage(app *ApplicationUsage) error {
	if app == nil || app.EndedAt != nil {
		return nil
	}

	// update the application with the ended_at
	endedAt := time.Now().Unix()
	durationSeconds := int(endedAt - app.StartedAt)

	app.EndedAt = &endedAt
	app.DurationSeconds = &durationSeconds

	if err := s.db.Save(&app).Error; err != nil {
		return fmt.Errorf("failed to update application usage: %w", err)
	}

	if s.UsageUpdates != nil {
		s.UsageUpdates <- app
	}

	return nil
}

func (s *Service) closeCurrentIdleUsage() error {
	var idleUsage ApplicationUsage
	err := s.db.Joins("Application").
		Where("application.name = ? AND application_usage.ended_at IS NULL", IdleApplicationName).
		Limit(1).Order("application_usage.started_at DESC").
		First(&idleUsage).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // Nothing to close
		}
		return fmt.Errorf("failed to find current idle usage: %w", err)
	}

	return s.closeApplicationUsage(&idleUsage)
}

func fetchFavicon(ctx context.Context, rawURL string) (string, error) {
	const googleFaviconURL = "https://www.google.com/s2/favicons?sz=64&domain="

	// Ensure URL has a scheme
	if !strings.HasPrefix(rawURL, "http") {
		rawURL = "http://" + rawURL
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleFaviconURL+parsedURL.Host, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch favicon: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch favicon: status code %d", response.StatusCode)
	}

	faviconData, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read favicon data: %w", err)
	}

	// Return base64 encoded favicon data
	return base64.StdEncoding.EncodeToString(faviconData), nil
}
