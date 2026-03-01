package serve

import (
	"context"
	"database/sql"
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
		mux.HandleFunc(geminiProxyPath, llmProxyHandler(gormDB, "gemini"))
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

// llmProxyHandler proxies requests to LLM APIs (e.g. Gemini) and enforces free-tier limits.
func llmProxyHandler(gormDB *gorm.DB, provider string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Authenticate Request
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")
		token = strings.TrimSpace(token)

		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		claims, err := api.ValidateToken(token)
		if err != nil {
			http.Error(w, "invalid or expired session", http.StatusUnauthorized)
			return
		}

		var user api.User
		if err := gormDB.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}

		// 2. Enforce Free Tier Limits (5 distractions per hour)
		if user.Tier == string(api.TierFree) {
			var count int64
			oneHourAgo := time.Now().Add(-1 * time.Hour).Unix()

			err := gormDB.Model(&api.LLMUsageLog{}).
				Where("user_id = ? AND provider = ? AND created_at >= ?", user.ID, provider, oneHourAgo).
				Count(&count).Error

			if err != nil {
				slog.Error("failed to check llm limit", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if count >= 5 {
				slog.Info("llm limit reached for free tier user", "user_id", user.ID, "count", count)
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
		}

		// 3. Build Target URL
		var targetURLStr string
		if provider == "gemini" {
			const geminiBaseURL = "https://generativelanguage.googleapis.com"
			targetPath := strings.TrimPrefix(r.URL.Path, "/api/v1/gemini")
			if targetPath == "" {
				targetPath = "/"
			}
			tURL, err := url.Parse(geminiBaseURL + targetPath)
			if err != nil {
				slog.Error("failed to parse target URL", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Preserve query parameters and append API key
			query := tURL.Query()
			if r.URL.RawQuery != "" {
				query, _ = url.ParseQuery(r.URL.RawQuery)
			}
			query.Set("key", os.Getenv("GEMINI_API_KEY"))
			tURL.RawQuery = query.Encode()
			targetURLStr = tURL.String()
		} else {
			http.Error(w, "unsupported provider", http.StatusBadRequest)
			return
		}

		slog.Info("proxying req to LLM", "provider", provider, "method", r.Method, "target", targetURLStr)

		proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURLStr, r.Body)
		if err != nil {
			slog.Error("failed to create proxy request", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		proxyReq.Header.Del("Connection")
		proxyReq.Header.Del("Keep-Alive")
		proxyReq.Header.Del("Proxy-Authenticate")
		proxyReq.Header.Del("Proxy-Authorization")
		proxyReq.Header.Del("Te")
		proxyReq.Header.Del("Trailers")
		proxyReq.Header.Del("Transfer-Encoding")
		proxyReq.Header.Del("Upgrade")

		// Execute proxy request
		client := &http.Client{Timeout: 120 * time.Second}
		resp, err := client.Do(proxyReq)
		if err != nil {
			slog.Error("failed to execute proxy request", "error", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		if resp.StatusCode >= 400 {
			slog.Error("proxy request failed", "status code", resp.StatusCode)
			w.WriteHeader(resp.StatusCode)
			if _, err := io.Copy(w, resp.Body); err != nil {
				slog.Error("failed to copy response body", "error", err)
			}
			return
		}

		// 4. Capture & Inspect Response Body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("failed to read response body", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Check if it's a distracting classification
		if user.Tier == string(api.TierFree) && strings.Contains(string(bodyBytes), `"classification": "distracting"`) || strings.Contains(string(bodyBytes), `"classification":"distracting"`) {
			logEntry := api.LLMUsageLog{
				UserID:    user.ID,
				Provider:  provider,
				CreatedAt: time.Now().Unix(),
			}
			if err := gormDB.Create(&logEntry).Error; err != nil {
				slog.Error("failed to insert llm usage log", "error", err)
				// Don't fail the request just because logging failed
			}
		}

		w.WriteHeader(resp.StatusCode)
		w.Write(bodyBytes)
	}
}
