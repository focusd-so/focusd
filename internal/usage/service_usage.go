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

	"github.com/focusd-so/focusd/internal/identity"
)

// IdleChanged is called when the idle state of the user changes (e.g. user starts or stops using the computer)
// When idle changes one of the following can happen:
//   - if user has been idle and idle triggers again, keep the current idle usage open to ensure idempotency
//   - if user has not been idle and idle triggers, close the current application usage and open a new idle usage
//   - if user has been idle and idle stops, no direct usage change is performed here; the next TitleChanged event closes idle
func (s *Service) IdleChanged(ctx context.Context, isIdle bool) (*ApplicationUsage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if isIdle {
		application, err := s.getOrCreateApplication(ctx, IdleApplicationName, "", nil, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get or create application: %w", err)
		}

		return s.usageChanged(ctx, application.NewUsage("", nil))
	}

	return nil, nil
}

// TitleChanged is called when the title of the current application changes,
// whether it's a new application or the same application title has changed
func (s *Service) TitleChanged(ctx context.Context, executablePath, windowTitle, appName, icon string, bundleID, browserURL, appCategory *string) (*ApplicationUsage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var normalizedBrowserURL *string
	if browserURL != nil {
		parsed, _ := parseURLNormalized(fromPtr(browserURL))
		normalizedBrowserURL = withPtr(parsed.String())
	}

	application, err := s.getOrCreateApplication(ctx, appName, icon, bundleID, normalizedBrowserURL, appCategory)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create application: %w", err)
	}

	usage, err := s.usageChanged(ctx, application.NewUsage(windowTitle, normalizedBrowserURL))

	return usage, err
}

func (s *Service) usageChanged(ctx context.Context, usage ApplicationUsage) (*ApplicationUsage, error) {
	currentApplicationUsage, err := s.getCurrentApplicationUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to find current application usage: %w", err)
	}

	// if the current application and new application usage are the same,
	// no additional action, let the current application usage continue
	if currentApplicationUsage.Same(usage) {
		usage = currentApplicationUsage
	} else {
		// if new application usage is detected close the current application usage before creating a new one
		if err := s.closeApplicationUsage(&currentApplicationUsage); err != nil {
			return nil, fmt.Errorf("failed to close current application usage: %w", err)
		}

		if err := s.saveApplicationUsage(&usage); err != nil {
			return nil, fmt.Errorf("failed to save application usage: %w", err)
		}
	}

	if usage.Application.Name == IdleApplicationName {
		return &usage, nil
	}

	classification, err := s.classifyApplicationUsage(ctx, &usage)
	if err != nil {
		errMsg := err.Error()
		usage.ClassificationError = &errMsg
		usage.Classification = ClassificationNone
	} else if classification != nil {
		usage.Classification = classification.Classification
		usage.ClassificationSource = &classification.ClassificationSource
		usage.ClassificationReasoning = &classification.Reasoning
		usage.ClassificationConfidence = &classification.ConfidenceScore

		classification.DetectedProject = fromPtr(usage.DetectedProject)
		classification.DetectedCommunicationChannel = fromPtr(usage.DetectedCommunicationChannel)

		usage.Tags = make([]ApplicationUsageTags, len(classification.Tags))
		for i, tag := range classification.Tags {
			usage.Tags[i] = ApplicationUsageTags{
				Tag: tag,
			}
		}

		usage.ClassificationSandboxContext = withPtr(classification.SandboxContext)
		usage.ClassificationSandboxResponse = classification.SandboxResponse
		usage.ClassificationSandboxLogs = withPtr(classification.SandboxLogs)
	}

	// calculate termination mode.
	enforcementAction, err := s.CalculateEnforcementDecision(ctx, &usage)
	if err != nil {
		termErr := err.Error()
		usage.EnforcementAction = EnforcementActionNone
		usage.EnforcementError = &termErr
	}

	usage.EnforcementAction = enforcementAction.Action
	usage.EnforcementReason = withPtr(enforcementAction.Reason)
	usage.EnforcementSource = withPtr(enforcementAction.Source)

	if err := s.db.Save(&usage).Error; err != nil {
		return nil, fmt.Errorf("failed to save application usage: %w", err)
	}

	if usage.EnforcementAction == EnforcementActionBlock {
		if err := s.closeApplicationUsage(&usage); err != nil {
			return nil, fmt.Errorf("failed to close application usage: %w", err)
		}
	}

	return &usage, s.saveApplicationUsage(&usage)
}

func (s *Service) saveApplicationUsage(applicationUsage *ApplicationUsage) error {
	if err := s.db.Save(applicationUsage).Error; err != nil {
		return fmt.Errorf("failed to save application usage: %w", err)
	}

	s.eventsMu.RLock()
	for _, fn := range s.onUsageUpdated {
		fn(applicationUsage)
	}
	s.eventsMu.RUnlock()

	return nil
}

func (s *Service) classifyApplicationUsage(ctx context.Context, applicationUsage *ApplicationUsage) (*ClassificationResponse, error) {
	// Do sandbox classification first, eg user defined custom rules
	customRulesResp, err := s.ClassifyCustomRules(ctx, WithAppNameContext(applicationUsage.Application.Name), WithWindowTitleContext(applicationUsage.WindowTitle), WithBrowserURLContext(fromPtr(applicationUsage.BrowserURL)))
	if err != nil {
		return nil, fmt.Errorf("failed to classify application usage with custom rules: %w", err)
	}

	tier := identity.GetAccountTier()
	isPaid := hasCustomRulesExecutionAccess(tier)

	if customRulesResp != nil && isPaid {
		return customRulesResp, nil
	}

	// Do obviously classification next, eg social media, news, shopping, etc.
	classification, err := s.classifyObviously(ctx, applicationUsage.Application.Name, applicationUsage.BrowserURL)
	if err != nil {
		if customRulesResp == nil {
			return nil, fmt.Errorf("failed to classify application usage with obviously: %w", err)
		}

		slog.Warn("failed to classify with obvious rules after custom rules; continuing to LLM fallback", "error", err)
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
	resp, err := s.ClassifyWithLLM(ctx, applicationUsage.Application.Name, applicationUsage.WindowTitle, applicationUsage.BrowserURL, applicationUsage.Application.BundleID, applicationUsage.Application.AppCategory)
	if err != nil {
		if customRulesResp != nil {
			fallbackResp := &ClassificationResponse{
				Classification:       ClassificationNone,
				ClassificationSource: ClassificationSourceObviously,
				Reasoning:            "Custom rules matched but fallback classification is unavailable",
				ConfidenceScore:      0,
				Tags:                 []string{"other"},
				SandboxContext:       customRulesResp.SandboxContext,
				SandboxResponse:      customRulesResp.SandboxResponse,
				SandboxLogs:          customRulesResp.SandboxLogs,
			}

			slog.Warn("failed to classify with LLM after custom rules; returning sandbox-backed fallback", "error", err)

			return fallbackResp, nil
		}

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
// The lookup identity is unified for both web and native applications:
//   - name + normalized hostname when rawURL resolves to a hostname
//   - name + hostname IS NULL for native applications
//
// For web applications, hostname and effective domain (TLD+1) are stored and a favicon
// is fetched when no icon is currently stored. For native applications, no hostname is set.
//
// Icon resolution order:
//   - keep existing stored icon when present
//   - for web apps, fallback to fetched favicon
//   - otherwise use provided icon
func (s *Service) getOrCreateApplication(ctx context.Context, name, icon string, bundleID, rawURL, appCategory *string) (Application, error) {
	var application Application
	var hostname *string

	if rawURL != nil {
		u, _ := parseURLNormalized(*rawURL)

		hostname = withPtr(u.Hostname())
	}

	query := s.db.Where("name = ?", name)
	if hostname != nil {
		query = query.Where("hostname = ?", *hostname)
	} else {
		query = query.Where("hostname IS NULL")
	}

	if err := query.First(&application).Error; err != nil {
		slog.Warn("failed to find application by identity", "error", err)
	}

	if application.ID == 0 {
		application = Application{Name: name, BundleID: bundleID}
	}

	if hostname != nil {
		application.Hostname = hostname

		// This normalizes subdomains: "mail.google.com" and "docs.google.com" both become "google.com".
		domain, _ := publicsuffix.EffectiveTLDPlusOne(*hostname)
		if domain != "" {
			application.Domain = &domain
		}

		if application.Icon == nil {
			appIcon, err := fetchFavicon(ctx, fmt.Sprintf("https://%s", *hostname))
			if err != nil {
				slog.Warn("failed to fetch app icon", "error", err)
			}

			if appIcon != "" {
				application.Icon = &appIcon
			}
		}
	}

	if fromPtr(application.Icon) == "" && icon != "" {
		application.Icon = &icon
	}

	application.AppCategory = appCategory

	if err := s.db.Save(&application).Error; err != nil {
		return Application{}, fmt.Errorf("failed to create application: %w", err)
	}

	return application, nil
}

func (s *Service) getCurrentApplicationUsage() (ApplicationUsage, error) {
	var application ApplicationUsage

	if err := s.db.Preload("Application").Preload("Tags").Where("ended_at IS NULL").Limit(1).Order("started_at DESC").First(&application).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return ApplicationUsage{}, fmt.Errorf("failed to find current application usage: %w", err)
		}
	}

	if application.ID == 0 {
		return ApplicationUsage{}, nil
	}

	return application, nil
}

func (s *Service) closeApplicationUsage(app *ApplicationUsage) error {
	if app.ID == 0 {
		return nil
	}

	if app.EndedAt != nil {
		return nil
	}

	endedAt := time.Now().Unix()
	durationSeconds := int(endedAt - app.StartedAt)

	app.EndedAt = &endedAt
	app.DurationSeconds = &durationSeconds

	if err := s.db.Save(&app).Error; err != nil {
		return fmt.Errorf("failed to update application usage: %w", err)
	}

	s.eventsMu.RLock()
	for _, fn := range s.onUsageUpdated {
		fn(app)
	}
	s.eventsMu.RUnlock()

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
