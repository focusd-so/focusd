package serve

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"github.com/urfave/cli/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/gen/api/v1/apiv1connect"
	"github.com/focusd-so/focusd/internal/api"
)

var Command = &cli.Command{
	Name: "serve",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "port",
			Value:   "8089",
			Usage:   "port to listen on",
			Aliases: []string{"p"},
			Sources: cli.EnvVars("PORT"),
		},
		&cli.StringFlag{
			Name:    "turso-db-url",
			Sources: cli.EnvVars("TURSO_CONNECTION_PATH"),
		},
		&cli.StringFlag{
			Name:    "turso-db-token",
			Sources: cli.EnvVars("TURSO_CONNECTION_TOKEN"),
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		gormDB, err := setupDatabase(cmd.String("turso-db-url"), cmd.String("turso-db-token"))
		if err != nil {
			return fmt.Errorf("failed to setup database: %w", err)
		}

		productIDs := map[apiv1.CheckoutProduct]string{
			apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PLUS: os.Getenv("POLAR_PRODUCT_PLUS_ID"),
			apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PRO:  os.Getenv("POLAR_PRODUCT_PRO_ID"),
		}

		apiService, err := api.NewServiceImpl(gormDB, productIDs)
		if err != nil {
			return fmt.Errorf("failed to create api service: %w", err)
		}

		mux := http.NewServeMux()

		apiPath, apiHandler := apiv1connect.NewApiServiceHandler(apiService, connect.WithInterceptors(
			api.NewAuthInterceptor(gormDB),
			validate.NewInterceptor(),
		))

		protocols := new(http.Protocols)
		protocols.SetHTTP1(true)
		protocols.SetUnencryptedHTTP2(true)
		mux.Handle(apiPath, apiHandler)

		// Polar.sh webhook endpoint
		// Configure POLAR_WEBHOOK_SECRET environment variable with the secret from Polar.sh dashboard
		webhookPath := "/api/v1/webhooks/polar"
		mux.HandleFunc(webhookPath, api.NewPolarWebhookHandler(apiService))
		slog.Info("serving polar webhook handler", "path", webhookPath)

		geminiProxyConfig := llmProxyConfig{
			Provider:   "gemini",
			BaseURL:    "https://generativelanguage.googleapis.com",
			PathPrefix: "/api/v1/gemini",
			SetupRequest: func(_ *http.Request, targetURL *url.URL, _ *http.Request) error {
				apiKey := os.Getenv("GEMINI_API_KEY")
				if apiKey == "" {
					return fmt.Errorf("missing GEMINI_API_KEY")
				}

				query := targetURL.Query()
				query.Set("key", apiKey)
				targetURL.RawQuery = query.Encode()
				return nil
			},
			ExtractUsage: extractGeminiUsageMetadata,
		}

		openAIProxyConfig := llmProxyConfig{
			Provider:   "openai",
			BaseURL:    "https://api.openai.com",
			PathPrefix: "/api/v1/openai",
			SetupRequest: func(_ *http.Request, _ *url.URL, proxyReq *http.Request) error {
				apiKey := os.Getenv("OPENAI_API_KEY")
				if apiKey == "" {
					return fmt.Errorf("missing OPENAI_API_KEY")
				}

				proxyReq.Header.Set("Authorization", "Bearer "+apiKey)
				return nil
			},
			ExtractUsage: extractOpenAIUsageMetadata,
		}

		anthropicProxyConfig := llmProxyConfig{
			Provider:   "anthropic",
			BaseURL:    "https://api.anthropic.com",
			PathPrefix: "/api/v1/anthropic",
			SetupRequest: func(_ *http.Request, _ *url.URL, proxyReq *http.Request) error {
				apiKey := os.Getenv("ANTHROPIC_API_KEY")
				if apiKey == "" {
					return fmt.Errorf("missing ANTHROPIC_API_KEY")
				}

				version := os.Getenv("ANTHROPIC_VERSION")
				if version == "" {
					version = "2023-06-01"
				}

				proxyReq.Header.Set("x-api-key", apiKey)
				proxyReq.Header.Set("anthropic-version", version)
				if proxyReq.Header.Get("Content-Type") == "" {
					proxyReq.Header.Set("Content-Type", "application/json")
				}
				return nil
			},
			ExtractUsage: extractAnthropicUsageMetadata,
		}

		grokProxyConfig := llmProxyConfig{
			Provider:   "grok",
			BaseURL:    "https://api.x.ai",
			PathPrefix: "/api/v1/grok",
			SetupRequest: func(_ *http.Request, _ *url.URL, proxyReq *http.Request) error {
				apiKey := os.Getenv("GROK_API_KEY")
				if apiKey == "" {
					return fmt.Errorf("missing GROK_API_KEY")
				}

				proxyReq.Header.Set("Authorization", "Bearer "+apiKey)
				return nil
			},
			ExtractUsage: extractGrokUsageMetadata,
		}

		// LLM API proxy endpoints
		geminiProxyPath := "/api/v1/gemini/"
		mux.HandleFunc(geminiProxyPath, newLLMProxyHandler(gormDB, geminiProxyConfig))
		slog.Info("serving gemini proxy handler", "path", geminiProxyPath)

		openAIProxyPath := "/api/v1/openai/"
		mux.HandleFunc(openAIProxyPath, newLLMProxyHandler(gormDB, openAIProxyConfig))
		slog.Info("serving openai proxy handler", "path", openAIProxyPath)

		anthropicProxyPath := "/api/v1/anthropic/"
		mux.HandleFunc(anthropicProxyPath, newLLMProxyHandler(gormDB, anthropicProxyConfig))
		slog.Info("serving anthropic proxy handler", "path", anthropicProxyPath)

		grokProxyPath := "/api/v1/grok/"
		mux.HandleFunc(grokProxyPath, newLLMProxyHandler(gormDB, grokProxyConfig))
		slog.Info("serving grok proxy handler", "path", grokProxyPath)

		slog.Info("serving rpc handler for api v1 service", "path", apiPath)

		h2Handler := h2c.NewHandler(mux, &http2.Server{})

		server := &http.Server{
			Addr:    ":" + cmd.String("port"),
			Handler: h2Handler, // Use the wrapped handler here
			// ReadHeaderTimeout is recommended to prevent Slowloris attacks
			ReadHeaderTimeout: 3 * time.Second,
			Protocols:         protocols,
		}

		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)

		go func() {
			slog.Info("serving http server for api v1 service", "addr", ":"+cmd.String("port"))
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("failed to serve engine service", "error", err)
				os.Exit(1)
			}
		}()

		<-sigint
		slog.Info("shutting down engine service")

		// Create a timeout context for shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("server forced to shutdown", "error", err)
		}

		slog.Info("engine service shut down")
		return nil
	},
}

func setupDatabase(url, token string) (*gorm.DB, error) {
	connStr := url
	if url == "" && token == "" {
		connStr = "file:focusd-api.db"
	} else if token != "" {
		connStr = fmt.Sprintf("%s?authToken=%s", url, token)
	}

	sqlDB, err := sql.Open("libsql", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open sql connection: %w", err)
	}

	gormDB, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open gorm connection: %w", err)
	}

	return gormDB, nil
}

// geminiProxyHandler proxies requests to Google's Generative Language API
// Requests to /api/v1/gemini/* are forwarded to https://generativelanguage.googleapis.com/*
type llmProxyConfig struct {
	Provider     string
	BaseURL      string
	PathPrefix   string
	SetupRequest func(incomingReq *http.Request, targetURL *url.URL, proxyReq *http.Request) error
	ExtractUsage func(body []byte) (input int, output int, total int)
}

// newLLMProxyHandler proxies requests to a configured upstream LLM provider.
func newLLMProxyHandler(db *gorm.DB, config llmProxyConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Fast-fail: Authenticate
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token == "" {
			http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		claims, err := api.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		// Rate Limiting
		var requestCount int64
		todayUnix := time.Now().Add(-24 * time.Hour).Unix()
		if err := db.Model(&api.LLMProxyUsage{}).Where("user_id = ? AND created_at >= ?", claims.UserID, todayUnix).Count(&requestCount).Error; err != nil {
			slog.Error("failed to query proxy usage count", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if requestCount >= 5000 {
			slog.Warn("user exceeded daily proxy limit", "user_id", claims.UserID)
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		// Strip the proxy prefix to get the target path
		targetPath := strings.TrimPrefix(r.URL.Path, config.PathPrefix)
		if targetPath == "" {
			targetPath = "/"
		}
		if !strings.HasPrefix(targetPath, "/") {
			targetPath = "/" + targetPath
		}

		// Build the target URL
		targetURL, err := url.Parse(config.BaseURL + targetPath)
		if err != nil {
			slog.Error("failed to parse target URL", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Preserve query parameters
		targetURL.RawQuery = r.URL.RawQuery

		slog.Info("proxying request to LLM API", "provider", config.Provider, "method", r.Method, "target", targetURL.String())

		// Create the proxy request
		// Create the proxy request using a detached context so the upstream request
		// to Google isn't canceled if the client disconnects mid-flight.
		proxyReq, err := http.NewRequestWithContext(context.WithoutCancel(r.Context()), r.Method, targetURL.String(), r.Body)
		if err != nil {
			slog.Error("failed to create proxy request", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Copy headers from original request
		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		// Remove hop-by-hop headers
		proxyReq.Header.Del("Connection")
		proxyReq.Header.Del("Keep-Alive")
		proxyReq.Header.Del("Proxy-Authenticate")
		proxyReq.Header.Del("Proxy-Authorization")
		proxyReq.Header.Del("Te")
		proxyReq.Header.Del("Trailers")
		proxyReq.Header.Del("Transfer-Encoding")
		proxyReq.Header.Del("Upgrade")
		proxyReq.Header.Del("Authorization")

		if config.SetupRequest != nil {
			if err := config.SetupRequest(r, targetURL, proxyReq); err != nil {
				slog.Error("failed to setup provider request", "provider", config.Provider, "error", err)
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
				return
			}
			proxyReq.URL = targetURL
		}

		// Execute the proxy request
		client := &http.Client{Timeout: 120 * time.Second}
		resp, err := client.Do(proxyReq)
		if err != nil {
			slog.Error("failed to execute proxy request", "error", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Copy the response body
		capturedBody, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("failed to read response body", "error", err)
		}

		// if the status code is anything >= 400, print an error
		if resp.StatusCode >= 400 {
			logBody := capturedBody
			if resp.Header.Get("Content-Encoding") == "gzip" {
				if gr, err := gzip.NewReader(bytes.NewReader(capturedBody)); err == nil {
					if decompressed, err := io.ReadAll(gr); err == nil {
						logBody = decompressed
					}
				}
			}
			slog.Error("proxy request failed", "status code", resp.StatusCode, "body", string(logBody))
		}

		// Set the status code
		w.WriteHeader(resp.StatusCode)

		// Synchronously parse tokens and save usage
		inputTokens, outputTokens, totalTokens := 0, 0, 0
		if config.ExtractUsage != nil {
			inputTokens, outputTokens, totalTokens = config.ExtractUsage(capturedBody)
		}

		usage := api.LLMProxyUsage{
			UserID:       claims.UserID,
			CreatedAt:    time.Now().Unix(),
			Provider:     config.Provider,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  totalTokens,
		}

		if err := db.Create(&usage).Error; err != nil {
			slog.Error("failed to save LLM proxy usage log", "error", err)
		}

		// Write the captured body back to the client
		if _, err := w.Write(capturedBody); err != nil {
			slog.Error("failed to write response body", "error", err)
		}
	}
}

func extractGeminiUsageMetadata(body []byte) (input int, output int, total int) {
	type geminiResponse struct {
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	// Try to parse the whole body as a single JSON object (non-streaming)
	var resp geminiResponse
	if err := json.Unmarshal(body, &resp); err == nil && resp.UsageMetadata.TotalTokenCount > 0 {
		return resp.UsageMetadata.PromptTokenCount, resp.UsageMetadata.CandidatesTokenCount, resp.UsageMetadata.TotalTokenCount
	}

	// If it failed or has 0 tokens, it might be an SSE stream.
	// SSE chunks start with "data: " and end with "\n\n"
	scanner := bufio.NewScanner(bytes.NewReader(body))
	// Some chunks might be large, so we expand the scanner buffer if needed
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			jsonStr := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if jsonStr == "" || jsonStr == "[DONE]" {
				continue
			}
			var streamResp geminiResponse
			if err := json.Unmarshal([]byte(jsonStr), &streamResp); err == nil {
				// Usage metadata is usually sent in the last chunk or cumulatively
				if streamResp.UsageMetadata.TotalTokenCount > total {
					input = streamResp.UsageMetadata.PromptTokenCount
					output = streamResp.UsageMetadata.CandidatesTokenCount
					total = streamResp.UsageMetadata.TotalTokenCount
				}
			}
		}
	}

	return input, output, total
}

func extractOpenAIUsageMetadata(body []byte) (input int, output int, total int) {
	type openAIResponse struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	// Try to parse the whole body as a single JSON object (non-streaming)
	var resp openAIResponse
	if err := json.Unmarshal(body, &resp); err == nil && resp.Usage.TotalTokens > 0 {
		return resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens
	}

	// If it failed or has 0 tokens, it might be an SSE stream.
	// SSE chunks start with "data: " and end with "\n\n"
	scanner := bufio.NewScanner(bytes.NewReader(body))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if after, ok := strings.CutPrefix(line, "data: "); ok {
			jsonStr := strings.TrimSpace(after)
			if jsonStr == "" || jsonStr == "[DONE]" {
				continue
			}
			var streamResp openAIResponse
			if err := json.Unmarshal([]byte(jsonStr), &streamResp); err == nil {
				if streamResp.Usage.TotalTokens > total {
					input = streamResp.Usage.PromptTokens
					output = streamResp.Usage.CompletionTokens
					total = streamResp.Usage.TotalTokens
				}
			}
		}
	}

	return input, output, total
}

func extractAnthropicUsageMetadata(body []byte) (input int, output int, total int) {
	type anthropicResponse struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	// Try to parse the whole body as a single JSON object (non-streaming)
	var resp anthropicResponse
	if err := json.Unmarshal(body, &resp); err == nil {
		totalTokens := resp.Usage.InputTokens + resp.Usage.OutputTokens
		if totalTokens > 0 {
			return resp.Usage.InputTokens, resp.Usage.OutputTokens, totalTokens
		}
	}

	// If it failed or has 0 tokens, it might be an SSE stream.
	scanner := bufio.NewScanner(bytes.NewReader(body))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			jsonStr := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if jsonStr == "" || jsonStr == "[DONE]" {
				continue
			}
			var streamResp anthropicResponse
			if err := json.Unmarshal([]byte(jsonStr), &streamResp); err == nil {
				streamTotal := streamResp.Usage.InputTokens + streamResp.Usage.OutputTokens
				if streamTotal > total {
					input = streamResp.Usage.InputTokens
					output = streamResp.Usage.OutputTokens
					total = streamTotal
				}
			}
		}
	}

	return input, output, total
}

func extractGrokUsageMetadata(body []byte) (input int, output int, total int) {
	return extractOpenAIUsageMetadata(body)
}
