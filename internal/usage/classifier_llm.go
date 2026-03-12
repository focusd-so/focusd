package usage

import "context"

func (s *Service) ClassifyWithLLM(ctx context.Context, appName, title string, url, bundleID, appCategory *string) (*ClassificationResponse, error) {
	if url != nil {
		return s.classifyWebsite(ctx, *url, title)
	}

	return s.classifyApplication(ctx, appName, title, bundleID, appCategory)
}
