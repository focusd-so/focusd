package extension

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type ConnectRequest struct {
	ApplicationName string `json:"application_name"`
}

type BootstrapResponse struct {
	WSURL   string `json:"ws_url"`
	APIKey  string `json:"api_key"`
	Version string `json:"version"`
}

// Hub tracks active websocket clients.
type extensionHub struct {
	mu       sync.RWMutex
	clients  map[string]*websocket.Conn // application name -> websocket connection
	upgrader websocket.Upgrader
}

var hub = &extensionHub{
	clients: make(map[string]*websocket.Conn),
	upgrader: websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	},
}

func RequireAPIKey(tokenProvider func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expected := tokenProvider()
			actual := strings.TrimSpace(r.URL.Query().Get("api_key"))

			if expected == "" || actual == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func BootstrapHandler(wsURL string, tokenProvider func() string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		response := BootstrapResponse{
			WSURL:   wsURL,
			APIKey:  tokenProvider(),
			Version: "1",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}

// Connect upgrades the request to websocket and tracks the client.
// When the connection is lost, the client is removed from the hub.
func Connect(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	request := ConnectRequest{
		ApplicationName: strings.TrimSpace(r.URL.Query().Get("application_name")),
	}

	if request.ApplicationName == "" {
		err := errors.New("application name is required")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, err
	}

	conn, err := hub.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	hub.mu.Lock()
	hub.clients[request.ApplicationName] = conn
	hub.mu.Unlock()

	go watch(request.ApplicationName, conn)

	return conn, nil
}

func HasClient(applicationName string) bool {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	_, ok := hub.clients[applicationName]
	return ok
}

// ClientCount returns the number of currently connected clients.
func ClientCount() int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.clients)
}

func watch(applicationName string, conn *websocket.Conn) {
	defer remove(applicationName, conn)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func remove(applicationName string, conn *websocket.Conn) {
	hub.mu.Lock()
	delete(hub.clients, applicationName)
	hub.mu.Unlock()
	_ = conn.Close()
}
