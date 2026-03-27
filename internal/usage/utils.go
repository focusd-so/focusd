package usage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	readability "codeberg.org/readeck/go-readability"
	"golang.org/x/net/html"
)

func parseURLNormalized(browserURL string) (*url.URL, error) {
	u, err := url.ParseRequestURI(browserURL)
	if err != nil {
		return nil, err
	}

	hostname := strings.ToLower(strings.TrimSpace(u.Hostname()))
	hostname = strings.TrimSuffix(hostname, ".")
	hostname = strings.TrimPrefix(hostname, "www.")

	u.Host = hostname

	return u, nil
}

type MetaData struct {
	Property string
	Content  string
}

func extractOpenGraph(httpClient *http.Client, url string) ([]MetaData, error) {
	response, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch open graph: %w", err)
	}
	defer response.Body.Close()

	var tags []MetaData
	z := html.NewTokenizer(response.Body)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if z.Err() == io.EOF {
				return tags, nil
			}
			return nil, z.Err()
		}

		t := z.Token()

		if t.Type == html.EndTagToken && t.Data == "head" {
			return tags, nil
		}

		if (t.Type == html.SelfClosingTagToken || t.Type == html.StartTagToken) && t.Data == "meta" {
			var prop, cont string
			for _, a := range t.Attr {
				switch a.Key {
				case "property":
					prop = a.Val
				case "content":
					cont = a.Val
				}
			}

			if strings.HasPrefix(prop, "og:") && cont != "" {
				tags = append(tags, MetaData{prop[len("og:"):], cont})
			}
		}
	}
}

func FetchMainContent(ctx context.Context, rawURL string) (string, error) {
	return fetchMainContent(ctx, rawURL)
}

func fetchMainContent(ctx context.Context, rawURL string) (string, error) {
	httpClient := &http.Client{Timeout: 5 * time.Second}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Focusd/1.0 (+https://github.com/focusd-so/focusd)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	parser := readability.NewParser()
	article, err := parser.Parse(resp.Body, parsedURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse content: %w", err)
	}

	return article.TextContent, nil
}

func createSandboxContext(appName string, url *string) sandboxContext {
	opts := []sandboxContextOption{WithAppNameContext(appName)}
	if url != nil {
		opts = append(opts, WithBrowserURLContext(*url))
	}

	return NewSandboxContext(opts...)
}

func withPtr[T any](v T) *T {
	// check if v is zero value
	if reflect.ValueOf(v).IsZero() {
		return nil
	}

	return &v
}

func fromPtr[T any](v *T) T {
	if v == nil {
		return *new(T)
	}

	return *v
}
