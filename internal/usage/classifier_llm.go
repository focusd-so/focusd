package usage

import "context"

func (s *Service) ClassifyWithLLM(ctx context.Context, appName, title, executablePath string, url *string) (*ClassificationResponse, error) {
	if url != nil {
		return s.classifyWebsite(ctx, *url, title)
	}

	return s.classifyApplication(ctx, appName, title, executablePath, url)
}
