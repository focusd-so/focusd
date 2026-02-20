package extension

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type ConnectRequest struct {
	ApplicationName string `json:"application_name"`
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

// Connect upgrades the request to websocket and tracks the client.
// When the connection is lost, the client is removed from the hub.
func Connect(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	var request ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}

	if request.ApplicationName == "" {
		return nil, errors.New("application name is required")
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
