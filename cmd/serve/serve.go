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
	"github.com/joho/godotenv"
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
		if err := godotenv.Load(); err != nil {
			return fmt.Errorf("failed to load .env file: %w", err)
		}

		gormDB, err := setupDatabase(cmd.String("turso-db-url"), cmd.String("turso-db-token"))
		if err != nil {
			return fmt.Errorf("failed to setup database: %w", err)
		}

		productIDs := map[apiv1.CheckoutProduct]string{
			apiv1.CheckoutProduct_CHECKOUT_PRODUCT_BASIC: os.Getenv("CHECKOUT_PRODUCT_BASIC_ID"),
			apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PRO:   os.Getenv("CHECKOUT_PRODUCT_PRO_ID"),
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

		// Gemini API proxy endpoint
		geminiProxyPath := "/api/v1/gemini/"
		mux.HandleFunc(geminiProxyPath, func(w http.ResponseWriter, r *http.Request) {
			geminiProxyHandler(w, r, gormDB)
		})
		slog.Info("serving gemini proxy handler", "path", geminiProxyPath)

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
func geminiProxyHandler(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	const geminiBaseURL = "https://generativelanguage.googleapis.com"

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
	targetPath := strings.TrimPrefix(r.URL.Path, "/api/v1/gemini")
	if targetPath == "" {
		targetPath = "/"
	}

	// Build the target URL
	targetURL, err := url.Parse(geminiBaseURL + targetPath)
	if err != nil {
		slog.Error("failed to parse target URL", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Preserve query parameters and append API key
	query := targetURL.Query()
	if r.URL.RawQuery != "" {
		query, _ = url.ParseQuery(r.URL.RawQuery)
	}
	query.Set("key", os.Getenv("GEMINI_API_KEY"))
	targetURL.RawQuery = query.Encode()

	slog.Info("proxying request to Gemini API", "method", r.Method, "target", targetURL.String())

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
	// Strip Authorization so the user's JWT isn't forwarded to Google —
	// authentication to Gemini is via the ?key= query param instead.
	proxyReq.Header.Del("Authorization")

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
	inputTokens, outputTokens, totalTokens := extractUsageMetadata(capturedBody)

	usage := api.LLMProxyUsage{
		UserID:       claims.UserID,
		CreatedAt:    time.Now().Unix(),
		Provider:     "gemini",
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

func extractUsageMetadata(body []byte) (input int, output int, total int) {
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
