package native

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const bootstrapURL = "http://127.0.0.1:50533/extension/bootstrap"

type hostRequest struct {
	Type            string `json:"type"`
	ApplicationName string `json:"application_name,omitempty"`
}

type hostResponse struct {
	Type            string `json:"type"`
	WSURL           string `json:"ws_url,omitempty"`
	APIKey          string `json:"api_key,omitempty"`
	ApplicationName string `json:"application_name,omitempty"`
	Version         string `json:"version,omitempty"`
	Error           string `json:"error,omitempty"`
}

type bootstrapResponse struct {
	WSURL   string `json:"ws_url"`
	APIKey  string `json:"api_key"`
	Version string `json:"version"`
}

func ServeHost() error {
	for {
		raw, err := readMessage(os.Stdin)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		resp := handleMessage(raw)
		if err := writeMessage(os.Stdout, resp); err != nil {
			return err
		}
	}
}

func handleMessage(raw []byte) hostResponse {
	var req hostRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return hostResponse{Type: "error", Error: "invalid request payload"}
	}

	requestType := strings.TrimSpace(req.Type)
	if requestType == "" {
		requestType = "get_connection_info"
	}

	switch requestType {
	case "get_connection_info":
		bootstrap, err := fetchBootstrap()
		if err != nil {
			return hostResponse{Type: "error", Error: err.Error()}
		}

		return hostResponse{
			Type:            "connection_info",
			WSURL:           bootstrap.WSURL,
			APIKey:          bootstrap.APIKey,
			ApplicationName: req.ApplicationName,
			Version:         bootstrap.Version,
		}
	default:
		return hostResponse{Type: "error", Error: "unsupported request type"}
	}
}

func fetchBootstrap() (*bootstrapResponse, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, bootstrapURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create bootstrap request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query focusd bootstrap endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bootstrap endpoint returned status %d", resp.StatusCode)
	}

	var parsed bootstrapResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode bootstrap response: %w", err)
	}

	if parsed.WSURL == "" || parsed.APIKey == "" {
		return nil, fmt.Errorf("bootstrap response missing required fields")
	}

	return &parsed, nil
}

func readMessage(r io.Reader) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	size := binary.LittleEndian.Uint32(header)
	if size == 0 {
		return nil, fmt.Errorf("empty native message")
	}

	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func writeMessage(w io.Writer, msg hostResponse) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(make([]byte, 0, 4+len(payload)))
	header := make([]byte, 4)
	binary.LittleEndian.PutUint32(header, uint32(len(payload)))
	buf.Write(header)
	buf.Write(payload)

	_, err = w.Write(buf.Bytes())
	return err
}
