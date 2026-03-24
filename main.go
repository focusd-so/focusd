package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/api"
	"github.com/focusd-so/focusd/internal/extension"
	"github.com/focusd-so/focusd/internal/identity"
	"github.com/focusd-so/focusd/internal/native"
	"github.com/focusd-so/focusd/internal/nativemessaging"
	"github.com/focusd-so/focusd/internal/settings"
	"github.com/focusd-so/focusd/internal/updater"
	"github.com/focusd-so/focusd/internal/usage"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

// Version and AppleTeamID are set at build time via -ldflags
var Version = "dev"
var AppleTeamID = ""

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/trayicon.png
var trayIcon []byte

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[*usage.ApplicationUsage]("usage:update")
	application.RegisterEvent[usage.ProtectionPause]("protection:status")
	application.RegisterEvent[usage.LLMDailySummary]("daily-summary:ready")
	application.RegisterEvent[any]("authctx:updated")
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	if isNativeMessagingHostMode() {
		if err := nativemessaging.ServeHost(); err != nil {
			log.Fatalf("failed to serve native messaging host: %v", err)
		}
		return
	}

	userdir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed to get user home directory: %v", err)
	}

	configDir := filepath.Join(userdir, ".focusd")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		log.Fatalf("failed to create config directory: %v", err)
	}

	// Configure logging
	logCloser, err := setupLogging()
	if err != nil {
		log.Printf("failed to setup logging: %v", err)
	} else if logCloser != nil {
		defer logCloser.Close()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	extensionSessionAPIKey, err := generateSessionAPIKey()
	if err != nil {
		log.Fatalf("failed to generate extension session API key: %v", err)
	}

	if err := nativemessaging.EnsureHostManifests(); err != nil {
		slog.Error("failed to ensure native messaging manifests", "error", err)
	}

	var (
		db = setupDB()
	)

	settingsService, err := settings.NewService(configDir)
	if err != nil {
		log.Fatal("failed to create settings service: %w", err)
	}

	// This client is used to perform the handshake and get the token. It is not authenticated.
	// It should only be used to perform the handshake and get the token.
	apiUntrustedClient := api.NewClient(settings.APIBaseURL())

	if err := identity.ScheduleHandshake(ctx, apiUntrustedClient); err != nil {
		slog.Error("failed to schedule handshake", "error", err)
	}

	apiAuthenticatedClient := api.NewClient(settings.APIBaseURL(), api.NewSigningInterceptor())

	identityService := identity.NewService(apiAuthenticatedClient)

	usageService, err := usage.NewService(ctx, db)
	if err != nil {
		log.Fatal("failed to create usage service: %w", err)
	}

	mux, _, err := setUpWebServer(ctx, extensionSessionAPIKey, usageService)
	if err != nil {
		log.Fatal("failed to setup web server: %w", err)
	}

	usageService.RegisterHTTPHandlers(mux)

	native.OnTitleChange(func(event native.NativeEvent) {
		hasClient := extension.HasClient(event.AppName)

		// an extension has been connected to handle app title changes and blocking for this app
		if hasClient {
			return
		}

		var (
			url      *string
			bundleID *string
			category *string
		)
		if event.URL != "" {
			url = &event.URL
		}
		if event.BundleID != "" {
			bundleID = &event.BundleID
		}
		if event.AppCategory != "" {
			category = &event.AppCategory
		}

		appUsage, err := usageService.TitleChanged(
			ctx,
			event.ExecutablePath,
			event.Title,
			event.AppName,
			event.Icon,
			bundleID,
			url,
			category,
		)
		if err != nil {
			slog.Error("failed to handle title change", "error", err)
			return
		}

		handleBlockedUsage(appUsage, event.AppName, event.Title, url)

	})

	var updaterService *updater.Service
	if settings.IsProductionBuild() {
		updaterService = updater.NewService(Version, "focusd-so", "focusd", AppleTeamID)
	}

	nativeService := native.NewNativeService()

	services := []application.Service{
		application.NewService(usageService),
		application.NewService(settingsService),
		application.NewService(identityService),
		application.NewService(nativeService),
	}
	if updaterService != nil {
		services = append(services, application.NewService(updaterService))
	}

	wailsApp := application.New(application.Options{
		Name:        "Focusd",
		Description: "Stay in flow, ship without distractions",
		Services:    services,
		LogLevel:    slog.LevelWarn,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	usageService.OnProtectionPause(func(pause usage.ProtectionPause) {
		wailsApp.Event.Emit("protection:status", pause)
	})

	usageService.OnProtectionResumed(func(pause usage.ProtectionPause) {
		wailsApp.Event.Emit("protection:status", pause)
	})

	usageService.OnLLMDailySummaryReady(func(summary usage.LLMDailySummary) {
		wailsApp.Event.Emit("daily-summary:ready", summary)

		// TODO: use proper system api to send a notification
		exec.Command("osascript", "-e",
			fmt.Sprintf(`display notification "%s" with title "Focusd" subtitle "Daily Summary"`,
				summary.Headline)).Run()
	})

	wailsApp.OnShutdown(cancel)

	usageService.OnUsageUpdated(func(appUsage *usage.ApplicationUsage) {
		if appUsage != nil {
			wailsApp.Event.Emit("usage:update", appUsage)
		}
	})

	native.OnIdleChange(func(idleSeconds float64) {
		usageService.IdleChanged(ctx, idleSeconds > 120)
	})

	if updaterService != nil {
		go updaterService.Start(ctx)
	}

	// Create the system tray
	systemTray := wailsApp.SystemTray.New()

	// Configure Tray
	// Set the tray icon
	systemTray.SetIcon(trayIcon)

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Window 1",
		Width:     1200,
		Height:    800,
		Frameless: true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHidden,
		},
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		URL:              "/",
	})

	// Hide the window once the webview has finished loading the Wails runtime.
	// We avoid using Hidden:true in window options because Wails v3 alpha dispatches
	// mac:WindowDidUpdate events to the webview before _wails is defined, causing
	// a TypeError infinite loop when the window starts hidden.
	var webviewReady bool
	window.OnWindowEvent(events.Mac.WebViewDidFinishNavigation, func(event *application.WindowEvent) {
		if !webviewReady {
			webviewReady = true
			window.Hide()
		}
	})

	// Register handler for protocol events
	wailsApp.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl, func(event *application.ApplicationEvent) {
		slog.Info("application opened with URL", "url", event.Context().URL())

		u, err := url.Parse(event.Context().URL())
		if err != nil {
			slog.Error("failed to parse URL", "error", err)
			return
		}

		switch u.Hostname() {
		case "checkout":
			if err := identity.PerformHandshake(ctx, apiUntrustedClient); err != nil {
				return
			}

			wailsApp.Event.Emit("authctx:updated", identity.GetAccountTier())
		}

		// toggle window open
		window.Show()
		window.Focus()
	})

	// Tray Click (Toggle)
	menu := wailsApp.NewMenu()
	if updaterService != nil {
		menu.Add("Check for Updates...").OnClick(func(_ *application.Context) {
			go func() {
				info, err := updaterService.CheckForUpdate(ctx)
				if err != nil {
					slog.Error("manual update check failed", "error", err)
					return
				}
				if info == nil {
					slog.Info("manual update check: already up to date")
					return
				}
				slog.Info("manual update check: update available, applying", "version", info.Version)
				updaterService.ApplyUpdate(ctx)
			}()
		})
	}
	menu.Add("Quit").OnClick(func(_ *application.Context) {
		wailsApp.Quit()
	})
	systemTray.SetMenu(menu)

	// Tray Click (Toggle)
	systemTray.OnClick(func() {
		if window.IsVisible() {
			wailsApp.Event.Emit("window:hidden", nil)
			window.Hide()
		} else {
			window.Show()
			window.Focus()
			wailsApp.Event.Emit("window:shown", nil)
		}
	})

	// If an error occurred while running the application, log it and exit.
	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

func setupLogging() (io.Closer, error) {
	var logPath string
	logName := "focusd.log"

	// Dev mode: if go.mod exists in current directory, we assume development.
	if _, err := os.Stat("go.mod"); err == nil {
		logPath = logName
	} else {
		configDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user config dir: %w", err)
		}
		appDir := filepath.Join(configDir, ".focusd")
		if err := os.MkdirAll(appDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create app config dir: %w", err)
		}
		logPath = filepath.Join(appDir, logName)
	}

	slog.Info("logging to", "path", logPath)

	// Rotating log writer: keeps 7 days of logs, rotates at 50 MB, compresses old files.
	rotator := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    200, // megabytes per file before rotation
		MaxAge:     7,   // days to retain old log files
		MaxBackups: 7,   // max number of old log files to keep
		Compress:   true,
	}

	// Create JSON handler backed by the rotating writer
	handler := slog.NewJSONHandler(rotator, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// Set as default logger
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return rotator, nil
}

func setupDB() *gorm.DB {
	dbName := "focusd.db"
	var dbPath string

	// Dev mode: if go.mod exists in current directory, we assume development.
	if _, err := os.Stat("go.mod"); err == nil {
		dbPath = dbName
	} else {
		configDir, err := os.UserConfigDir()
		if err != nil {
			log.Fatal("failed to get user config dir: %w", err)
		}
		appDir := filepath.Join(configDir, "focusd")
		if err := os.MkdirAll(appDir, 0755); err != nil {
			log.Fatal("failed to create app config dir: %w", err)
		}
		dbPath = filepath.Join(appDir, dbName)
	}

	connStr := "file:" + dbPath

	slog.Info("initialising database", "path", dbPath)

	sqlDB, err := sql.Open("libsql", connStr)
	if err != nil {
		log.Fatal("failed to open sql connection: %w", err)
	}

	gormDB, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		log.Fatal("failed to open gorm connection: %w", err)
	}

	return gormDB
}

type extensionEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type pageTitleChangedPayload struct {
	TabID     int     `json:"tabId"`
	Title     string  `json:"title"`
	WindowID  int     `json:"windowId"`
	URL       *string `json:"url"`
	Timestamp string  `json:"timestamp"`
}

type pageTitleClassifiedPayload struct {
	TabID int                     `json:"tabId"`
	Title string                  `json:"title"`
	Usage *usage.ApplicationUsage `json:"usage"`
}

type pageTitleErrorPayload struct {
	TabID int    `json:"tabId"`
	Error string `json:"error"`
}

func handleExtensionMessage(ctx context.Context, usageService *usage.Service, applicationName string, conn *websocket.Conn, payload []byte) error {
	var envelope extensionEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return writePageTitleError(conn, 0, fmt.Sprintf("invalid message payload: %v", err))
	}

	switch envelope.Type {
	case "page_title_changed":
		var message pageTitleChangedPayload
		if err := json.Unmarshal(envelope.Payload, &message); err != nil {
			return writePageTitleError(conn, 0, fmt.Sprintf("invalid page_title_changed payload: %v", err))
		}

		if message.Title == "" {
			return writePageTitleError(conn, message.TabID, "title is required")
		}

		var browserURL *string
		if message.URL != nil && *message.URL != "" {
			browserURL = message.URL
		}

		appUsage, err := usageService.TitleChanged(
			ctx,
			applicationName,
			message.Title,
			applicationName,
			"",
			nil,
			browserURL,
			nil,
		)
		if err != nil {
			slog.Error("failed to handle extension page title change", "error", err, "application_name", applicationName, "tab_id", message.TabID)
			return writePageTitleError(conn, message.TabID, err.Error())
		}

		handleBlockedUsage(appUsage, applicationName, message.Title, browserURL)

		return conn.WriteJSON(map[string]any{
			"type": "page_title_classified",
			"payload": pageTitleClassifiedPayload{
				TabID: message.TabID,
				Title: message.Title,
				Usage: appUsage,
			},
		})
	default:
		return nil
	}
}

func writePageTitleError(conn *websocket.Conn, tabID int, errMsg string) error {
	return conn.WriteJSON(map[string]any{
		"type": "page_title_error",
		"payload": pageTitleErrorPayload{
			TabID: tabID,
			Error: errMsg,
		},
	})
}

func handleBlockedUsage(appUsage *usage.ApplicationUsage, appName, title string, browserURL *string) {
	if appUsage == nil || appUsage.EnforcementAction != usage.EnforcementActionBlock {
		return
	}

	tags := usage.ApplicationTagsSlice(appUsage.Tags).Tags()
	reasoning := ""
	if appUsage.ClassificationReasoning != nil {
		reasoning = *appUsage.ClassificationReasoning
	}

	if browserURL != nil {
		slog.Info("browser url provided, blocking url", "url", *browserURL)
		if err := native.BlockURL(*browserURL, title, reasoning, tags, appName); err != nil {
			slog.Error("failed to block URL", "url", *browserURL, "error", err)
			return
		}
	}

	if err := native.BlockApp(appName, title, reasoning, tags); err != nil {
		slog.Error("failed to block app", "appName", appName, "error", err)
	}
}

func setUpWebServer(ctx context.Context, extensionSessionAPIKey string, usageService *usage.Service) (*chi.Mux, int, error) {
	const port = 50533

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	extensionWSURL := fmt.Sprintf("ws://127.0.0.1:%d/extension/ws", port)
	tokenProvider := func() string {
		return extensionSessionAPIKey
	}

	r.Route("/extension", func(r chi.Router) {
		r.Get("/bootstrap", extension.BootstrapHandler(extensionWSURL, tokenProvider))
		r.With(extension.RequireAPIKey(tokenProvider)).Get("/ws", func(w http.ResponseWriter, req *http.Request) {
			if _, err := extension.Connect(w, req, func(applicationName string, conn *websocket.Conn, payload []byte) error {
				return handleExtensionMessage(ctx, usageService, applicationName, conn, payload)
			}); err != nil {
				slog.Warn("extension websocket connection failed", "error", err)
			}
		})
	})

	slog.Info("web server running on port", "port", port)

	server := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf(":%d", port),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("web server failed", "error", err)
			return
		}
	}()

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(ctx); err != nil {
			slog.Error("web server shutdown failed", "error", err)
			return
		}
	}()

	return r, port, nil
}

func isNativeMessagingHostMode() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--native-messaging-host" {
			return true
		}
	}

	return false
}

func generateSessionAPIKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
