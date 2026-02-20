package main

import (
	"context"
	"database/sql"
	"embed"
	_ "embed"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"google.golang.org/genai"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/api"
	"github.com/focusd-so/focusd/internal/extension"
	"github.com/focusd-so/focusd/internal/identity"
	"github.com/focusd-so/focusd/internal/native"
	"github.com/focusd-so/focusd/internal/settings"
	"github.com/focusd-so/focusd/internal/usage"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/trayicon.png
var trayIcon []byte

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[usage.ApplicationUsage]("usage:update")
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	// Configure logging
	logCloser, err := setupLogging()
	if err != nil {
		log.Printf("failed to setup logging: %v", err)
	} else if logCloser != nil {
		defer logCloser.Close()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		db               = setupDB()
		protectionPaused = make(chan bool)
	)

	mux, _, err := setUpWebServer(ctx)
	if err != nil {
		log.Fatal("failed to setup web server: %w", err)
	}

	settingsService, err := settings.NewService(db)
	if err != nil {
		log.Fatal("failed to create settings service: %w", err)
	}

	// Resolve the API base URL based on build type.
	apiBaseURL := "http://localhost:8089"
	if isProductionBuild {
		apiBaseURL = "https://focusd.so"
	}

	// This client is used to perform the handshake and get the token. It is not authenticated.
	// It should only be used to perform the handshake and get the token.
	apiUntrustedClient := api.NewClient(apiBaseURL)

	if err := identity.ScheduleHandshake(ctx, apiUntrustedClient); err != nil {
		slog.Error("failed to schedule handshake: %w", err)
	}

	apiAuthenticatedClient := api.NewClient(apiBaseURL, api.NewSigningInterceptor())

	identityService := identity.NewService(apiAuthenticatedClient)

	genaiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		HTTPOptions: genai.HTTPOptions{
			BaseURL: apiBaseURL + "/api/v1/gemini",
		},
	})

	usageService, err := usage.NewService(
		ctx, db, usage.WithProtectionPaused(func(pause usage.ProtectionPause) {
			slog.Info("protection has been paused", "reason", pause.Reason)

			protectionPaused <- true
		}),
		usage.WithProtectionResumed(func(pause usage.ProtectionPause) {
			slog.Info("protection has been resumed", "reason", pause.Reason)

			protectionPaused <- false
		}),
		usage.WithAppBlocker(func(appName, title, reason string, tags []string, browserURL *string) {
			client := extension.HasClient(appName)

			// if an extension has been connected to handle app, they should take care of blocking the app
			if client {
				return
			}

			if browserURL != nil {
				slog.Info("browser url provided, blocking url", "url", *browserURL)
				if err := native.BlockURL(*browserURL, title, reason, tags, appName); err != nil {
					slog.Error("failed to block URL", "url", *browserURL, "error", err)

					return
				}
			}

			slog.Info("no browser url provided, blocking app", "appName", appName)
			if err := native.BlockApp(appName, title, reason, tags); err != nil {
				slog.Error("failed to block app", "appName", appName, "error", err)

				return
			}
		}),
		usage.WithGenaiClient(genaiClient),
	)
	if err != nil {
		log.Fatal("failed to create usage service: %w", err)
	}

	usageService.RegisterHTTPHandlers(mux)

	native.OnTitleChange(func(event native.NativeEvent) {
		hasClient := extension.HasClient(event.AppName)

		// an extension has been connected to handle app title changes and blocking for this app
		if hasClient {
			return
		}

		err := usageService.TitleChanged(
			ctx,
			event.ExecutablePath,
			event.Title,
			event.AppName,
			event.Icon,
			&event.BundleID,
			&event.URL,
		)
		if err != nil {
			slog.Error("failed to handle title change", "error", err)
		}
	})

	wailsApp := application.New(application.Options{
		Name:        "Focusd",
		Description: "Stay in flow, ship without distractions",
		Services: []application.Service{
			application.NewService(usageService),
			application.NewService(settingsService),
			application.NewService(identityService),
		},
		LogLevel: slog.LevelWarn,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	wailsApp.OnShutdown(cancel)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case usage := <-usageService.UsageUpdates:
				wailsApp.Event.Emit("usage:update", *usage)
			}
		}
	}()

	native.OnTitleChange(func(event native.NativeEvent) {
		var bundleID *string
		if event.BundleID != "" {
			bundleID = &event.BundleID
		}

		var urlPtr *string
		if event.URL != "" {
			urlPtr = &event.URL
		}

		usageService.TitleChanged(ctx, event.ExecutablePath, event.Title, event.AppName, event.Icon, bundleID, urlPtr)
	})
	native.OnIdleChange(func(idleSeconds float64) {
		// usageService.IdleChanged(ctx, idleSeconds > 120)
	})
	go native.StartObserver()

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

			wailsApp.Event.Emit("identity:changed", nil)
		}

		// toggle window open
		window.Show()
		window.Focus()
	})

	// Drain the protectionPaused channel without calling SetTitle.
	// SetTitle triggers mac:WindowDidUpdate events which cause errors
	// in Wails v3 alpha when the webview runtime isn't ready.
	// The window is frameless with a hidden title bar, so the title
	// isn't visible anyway.
	go func() {
		for {
			select {
			case <-protectionPaused:
				// Protection state change received; no window title update needed
				// since the window is frameless with a hidden title bar.
			case <-ctx.Done():
				return
			}
		}
	}()

	// Tray Menu
	menu := wailsApp.NewMenu()
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
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
		configDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user config dir: %w", err)
		}
		appDir := filepath.Join(configDir, "focusd")
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

func setUpWebServer(ctx context.Context) (*chi.Mux, int, error) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// run on a random available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	slog.Info("web server running on port", "port", port)

	server := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf(":%d", port),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return
		}
	}()

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(ctx); err != nil {
			return
		}
	}()

	return r, port, nil
}
