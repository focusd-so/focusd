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

	"github.com/focusd-so/focusd/internal/timeline"
	"github.com/google/uuid"
)

type ApplicationUsagePayload struct {
	BasicClassificationResult

	ApplicationID int64 `json:"application_id" gorm:"not null"`

	WindowTitle string `json:"window_title" gorm:"not null"`
	BrowserURL  string `json:"browser_url,omitempty"`

	ClassificationSource ClassificationSource  `json:"classification_source" gorm:"index:idx_classification_source"`
	ClassificationResult *ClassificationResult `json:"classification_result,omitempty"`

	EnforcementResult EnforcementResult `json:"enforcement_result,omitempty"`
}

func (s *Service) IdleChanged(ctx context.Context, isIdle bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get the current active usage or idle event
	event, err := s.timelineService.GetActiveEventOfTypes(EventTypeUsageChanged, EventTypeUserIdleChanged)
	if err != nil {
		return err
	}

	// If no active event exists, we can't determine current state; normally there should be an active UsageChanged event.
	if event == nil {
		return nil
	}

	if isIdle {
		// If we are already in idle state, do nothing (idempotency)
		if event.Type == EventTypeUserIdleChanged {
			return nil
		}
		// Transition from active usage to idle: finish current usage and start idle
		if err := s.timelineService.EventFinished(event); err != nil {
			return err
		}
		_, err = s.timelineService.CreateEvent(EventTypeUserIdleChanged)
		return err
	}

	// User is now active (!isIdle)
	// If we were already active (UsageChanged), do nothing
	if event.Type == EventTypeUsageChanged {
		return nil
	}

	// Transition from idle back to active: just finish the idle event
	// Note: The next window activity will naturally start a new UsageChanged event
	return s.timelineService.EventFinished(event)
}

// TitleChanged is called when the title of the current application changes,
// whether it's a new application or the same application title has changed
func (s *Service) TitleChanged(ctx context.Context, appName, windowTitle, icon string, browserURL, appCategory *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizedURL, _, _ := parseURLNormalized(browserURL)

	application, err := s.getOrCreateApplication(ctx, appName, icon, normalizedURL, appCategory)
	if err != nil {
		return fmt.Errorf("failed to get or create application: %w", err)
	}

	usageKeyUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("app:%s,window:%s,url:%s", application.Name, windowTitle, fromPtr(browserURL))))

	lastEvent, err := s.timelineService.GetActiveEventOfTypes(EventTypeUsageChanged, EventTypeUserIdleChanged)
	if err != nil {
		slog.Warn("failed to get last event", "error", err)
	}

	if lastEvent != nil {
		if fromPtr(lastEvent.Key) == usageKeyUUID.String() && usageKeyUUID.String() != "" {
			return nil
		}
		if err := s.timelineService.EventFinished(lastEvent); err != nil {
			return fmt.Errorf("failed to finish last event: %w", err)
		}
	}

	payload := ApplicationUsagePayload{WindowTitle: windowTitle, ApplicationID: application.ID}
	if normalizedURL != nil {
		payload.BrowserURL = normalizedURL.String()
	}

	event, err := s.timelineService.CreateEvent(
		EventTypeUsageChanged,
		timeline.WithKey(usageKeyUUID.String()),
		timeline.WithPayload(payload),
	)
	if err != nil {
		return fmt.Errorf("creating usage event: %w", err)
	}

	classificationResult, err := s.classifyApplicationUsage(ctx, appName, windowTitle, normalizedURL, appCategory)
	if err != nil {
		return err
	}

	payload.ClassificationResult = classificationResult
	payload.Classification = classificationResult.Classification()
	payload.ClassificationSource = classificationResult.ClassificationSource()
	payload.ClassificationReason = classificationResult.ClassificationReason()

	for _, tag := range classificationResult.Tags() {
		event.Tags = append(event.Tags, timeline.NewTag(tag, TagTypeClassificationTag))
	}

	event.Tags = append(event.Tags, timeline.NewTag(string(classificationResult.Classification()), TagTypeClassification))

	if err := s.timelineService.UpdateEvent(&event, timeline.WithPayload(payload)); err != nil {
		return err
	}

	enforcementResult, err := s.CalculateEnforcement(ctx, appName, windowTitle, normalizedURL, classificationResult.Classification())
	if err != nil {
		return err
	}

	payload.EnforcementResult = enforcementResult
	if err := s.timelineService.UpdateEvent(&event, timeline.WithPayload(payload)); err != nil {
		return err
	}

	return err
}

func (s *Service) classifyApplicationUsage(ctx context.Context, name, windowTitle string, browserURL *url.URL, appCategory *string) (*ClassificationResult, error) {
	// Do sandbox classification first, eg user defined custom rules
	customRulesClassificationResult, err := s.ClassifyCustomRules(ctx, WithAppNameContext(name), WithWindowTitleContext(windowTitle), WithBrowserURLContext(browserURL))
	if err != nil {
		return nil, fmt.Errorf("failed to classify application usage with custom rules: %w", err)
	}

	// Do obviously classification next, eg social media, news, shopping, etc.
	obviClassificationResult, err := s.classifyObviously(ctx, name, browserURL)
	if err != nil {
		slog.Warn("failed to classify with obviously rules after custom rules; continuing to LLM fallback", "error", err)
	}

	if obviClassificationResult != nil {
		return &ClassificationResult{
			CustomRulesClassificationResult: customRulesClassificationResult,
			ObviouslyClassificationResult:   obviClassificationResult,
		}, nil
	}

	llmClassificationResult, err := s.ClassifyWithLLM(ctx, name, windowTitle, browserURL, appCategory)
	if err != nil {
		return nil, err
	}

	return &ClassificationResult{
		CustomRulesClassificationResult: customRulesClassificationResult,
		ObviouslyClassificationResult:   obviClassificationResult,
		LLMClassificationResult:         llmClassificationResult,
	}, nil
}

func (s *Service) getOrCreateApplication(ctx context.Context, name, icon string, browserURL *url.URL, appCategory *string) (Application, error) {
	var application Application

	if browserURL != nil {
		name = browserURL.Hostname()
	}

	if err := s.db.Where("name = ?", name).First(&application).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return Application{}, fmt.Errorf("failed to find application by identity: %w", err)
		}
	}

	if application.ID == 0 {
		application = Application{Name: name}
	}

	if browserURL != nil {
		domain, _ := publicsuffix.EffectiveTLDPlusOne(browserURL.Hostname())
		application.Domain = &domain

		if fromPtr(application.Icon) == "" {
			appIcon, err := fetchFavicon(ctx, fmt.Sprintf("https://%s", name))
			if err != nil {
				slog.Warn("failed to fetch app icon", "error", err)
			}

			application.Icon = &appIcon
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

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, googleFaviconURL+parsedURL.Host, nil)
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
