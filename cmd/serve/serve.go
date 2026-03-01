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
		mux.HandleFunc(geminiProxyPath, geminiProxyHandler)
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
func geminiProxyHandler(w http.ResponseWriter, r *http.Request) {
	const geminiBaseURL = "https://generativelanguage.googleapis.com"

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
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
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

	// if the status code is anything >= 400, print an error
	if resp.StatusCode >= 400 {
		slog.Error("proxy request failed", "status code", resp.StatusCode)
	}

	// Set the status code
	w.WriteHeader(resp.StatusCode)

	// Copy the response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		slog.Error("failed to copy response body", "error", err)
	}
}
