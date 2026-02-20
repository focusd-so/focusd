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
)

// IdleChanged is called when the idle state of the user changes (e.g. user starts or stops using the computer)
func (s *Service) IdleChanged(ctx context.Context, isIdle bool) error {
	if isIdle {
		if err := s.closeCurrentApplicationUsage(); err != nil {
			return fmt.Errorf("failed to close current application usage: %w", err)
		}

		// check if there is a current idle period
		var idlePeriod *IdlePeriod
		err := s.db.Where("ended_at IS NULL").Limit(1).Order("started_at desc").First(&idlePeriod).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("failed to find current idle period: %w", err)
			}
		}

		// if there is a current idle period, let it continue
		if idlePeriod != nil && idlePeriod.ID > 0 {
			return nil
		}

		// create a new idle period
		idlePeriod = &IdlePeriod{
			StartedAt: time.Now().Unix(),
			Reason:    "user_idle",
		}
		if err := s.db.Create(&idlePeriod).Error; err != nil {
			return fmt.Errorf("failed to create new idle period: %w", err)
		}

	} else {
		// close the current idle period
		if err := s.closeCurrentIdlePeriod(); err != nil {
			return fmt.Errorf("failed to close current idle period: %w", err)
		}
	}

	return nil
}

// TitleChanged is called when the title of the current application changes,
// whether it's a new application or the same application title has changed
func (s *Service) TitleChanged(ctx context.Context, executablePath, windowTitle, appName, icon string, bundleID, url *string) error {
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

	application, err := s.getOrCreateApplication(ctx, executablePath, appName, icon, bundleID, url)
	if err != nil {
		return fmt.Errorf("failed to get or create application: %w", err)
	}

	applicationUsage := ApplicationUsage{
		WindowTitle: windowTitle,
		BrowserURL:  url,
		StartedAt:   time.Now().Unix(),

		Application: application,
	}

	// save the application usage
	if err := s.db.Save(&applicationUsage).Error; err != nil {
		return fmt.Errorf("failed to save application usage: %w", err)
	}

	classification, err := s.classifyApplicationUsage(ctx, &applicationUsage)
	if err != nil {
		errMsg := err.Error()
		applicationUsage.ClassificationError = &errMsg
		applicationUsage.Classification = ClassificationError
	} else if classification != nil {
		applicationUsage.Classification = classification.Classification
		applicationUsage.ClassificationSource = classification.ClassificationSource
		applicationUsage.ClassificationReasoning = classification.Reasoning
		applicationUsage.ClassificationConfidence = classification.ConfidenceScore

		applicationUsage.DetectedProject = classification.DetectedProject
		applicationUsage.DetectedCommunicationChannel = classification.DetectedCommunicationChannel
		applicationUsage.Tags = make([]ApplicationUsageTags, len(classification.Tags))
		for i, tag := range classification.Tags {
			applicationUsage.Tags[i] = ApplicationUsageTags{
				Tag: tag,
			}
		}
	}

	// calculate termination mode.
	terminationMode, err := s.CalculateTerminationMode(ctx, &applicationUsage)
	if err != nil {
		applicationUsage.TerminationMode = TerminationModeNone
		applicationUsage.TerminationError = err.Error()
	}

	applicationUsage.TerminationMode = terminationMode.Mode
	applicationUsage.TerminationReasoning = terminationMode.Reasoning
	applicationUsage.TerminationSource = terminationMode.Source

	if err := s.db.Save(&applicationUsage).Error; err != nil {
		return fmt.Errorf("failed to save application usage: %w", err)
	}

	if applicationUsage.TerminationMode == TerminationModeBlock {
		s.appBlocker(applicationUsage.Application.Name, applicationUsage.WindowTitle, applicationUsage.TerminationReasoning, classification.Tags, applicationUsage.BrowserURL)
	}

	if s.UsageUpdates != nil {
		s.UsageUpdates <- &applicationUsage
	}

	return nil
}

func (s *Service) classifyApplicationUsage(ctx context.Context, applicationUsage *ApplicationUsage) (*ClassificationResponse, error) {
	// Do sandbox classification first, eg user defined custom rules
	classification, err := s.ClassifyCustomRules(ctx, applicationUsage.Application.Name, applicationUsage.Application.ExecutablePath, applicationUsage.BrowserURL)
	if err != nil {
		return nil, fmt.Errorf("failed to classify application usage with custom rules: %w", err)
	}

	if classification != nil {
		return classification, nil
	}

	// Do obviously classification next, eg social media, news, shopping, etc.
	classification, err = s.classifyObviously(ctx, applicationUsage.Application.Name, applicationUsage.Application.ExecutablePath, applicationUsage.BrowserURL)
	if err != nil {
		return nil, fmt.Errorf("failed to classify application usage with obviously: %w", err)
	}

	if classification != nil {
		return classification, nil
	}

	slog.Info("classifying application usage with LLM")
	resp, err := s.ClassifyWithLLM(ctx, applicationUsage.Application.Name, applicationUsage.WindowTitle, applicationUsage.Application.ExecutablePath, applicationUsage.BrowserURL)
	if err != nil {
		return nil, fmt.Errorf("failed to classify application usage with LLM: %w", err)
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
//   - executablePath: Full path to the application executable (e.g., "/Applications/Safari.app")
//   - name: Display name of the application (e.g., "Safari", "Google Chrome")
//   - icon: Base64-encoded icon data for native apps (ignored for web apps)
//   - bundleID: Optional macOS bundle identifier (e.g., "com.apple.Safari")
//   - rawURL: Optional URL for web applications; if nil, treated as native app
//
// Returns:
//   - Application: The found or newly created application record
//   - error: Any error encountered during database operations or favicon fetching
func (s *Service) getOrCreateApplication(ctx context.Context, executablePath, name, icon string, bundleID, rawURL *string) (Application, error) {
	// Handle web applications (browser tabs with URLs)
	if rawURL != nil {
		// Extract hostname from the URL (e.g., "www.google.com" from "https://www.google.com/search?q=...")
		u, err := url.Parse(*rawURL)
		if err != nil {
			slog.Warn("failed to parse URL", "error", err)
		}

		hostname := u.Hostname()
		if hostname == "" {
			slog.Warn("empty hostname")
		}

		// Attempt to find an existing application record by hostname.
		// Web apps are uniquely identified by hostname, so all tabs from the same
		// site (e.g., multiple Google tabs) share the same Application record.
		var application Application
		if err := s.db.Where("hostname = ?", hostname).First(&application).Error; err != nil {
			slog.Warn("failed to find application by hostname", "error", err)
		}

		// If no existing application found, create a new one with the provided metadata
		if application.ID == 0 {
			application = Application{
				Name:           name,
				ExecutablePath: executablePath,
				Hostname:       &hostname,
				BundleID:       bundleID,
			}
		}

		// Extract the effective domain (TLD+1) if not already set.
		// This normalizes subdomains: "mail.google.com" and "docs.google.com" both become "google.com".
		// Useful for grouping related sites and aggregating usage statistics.
		if application.Domain == nil {
			domain, _ := publicsuffix.EffectiveTLDPlusOne(hostname)
			if domain == "" {
				slog.Warn("empty domain")
			}

			application.Domain = &domain
		}

		// Fetch favicon if icon is empty or very small (old 16x16 ICO format, typically <500 chars base64)
		if len(application.Icon) < 500 {
			appIcon, err := fetchFavicon(ctx, fmt.Sprintf("https://%s", hostname))
			if err != nil {
				slog.Warn("failed to fetch app icon", "error", err)
			}

			if appIcon != "" {
				application.Icon = appIcon
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
	if err := s.db.Where("executable_path = ?", executablePath).First(&application).Error; err != nil {
		slog.Warn("failed to find application by executable path", "error", err)
	}

	// If no existing application found, create a new one with the provided metadata.
	// For native apps, the icon is provided by the OS (e.g., from the app bundle).
	if application.ID == 0 {
		application = Application{
			Name:           name,
			ExecutablePath: executablePath,
			BundleID:       bundleID,
			Icon:           icon,
		}
	} else if application.Icon == "" && icon != "" {
		// Update the icon for existing apps that don't have one yet
		application.Icon = icon
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

func (s *Service) closeCurrentApplicationUsage() error {
	application, err := s.getCurrentApplicationUsage()
	if err != nil {
		return fmt.Errorf("failed to find current application usage: %w", err)
	}

	if application == nil {
		return nil
	}

	return s.closeApplicationUsage(application)
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

	return nil
}

func (s *Service) closeCurrentIdlePeriod() error {
	var idlePeriod IdlePeriod
	if err := s.db.Where("ended_at IS NULL").Limit(1).Order("started_at desc").First(&idlePeriod).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to find current idle period: %w", err)
		}
	}

	if idlePeriod.EndedAt != nil {
		return nil
	}

	now := time.Now().Unix()

	idlePeriod.EndedAt = &now

	durationInSeconds := int(now - idlePeriod.StartedAt)
	idlePeriod.DurationSeconds = &durationInSeconds

	if err := s.db.Save(&idlePeriod).Error; err != nil {
		return fmt.Errorf("failed to update idle period: %w", err)
	}

	return nil
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
