package api

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"

	"github.com/focusd-so/focusd/gen/api/v1/apiv1connect"
)

// NewClient creates a new ApiServiceClient with signing/authentication handling.
// The signing interceptor handles token management and automatic refresh on expiry.
func NewClient(baseURL string, interceptors ...connect.Interceptor) apiv1connect.ApiServiceClient {

	client := apiv1connect.NewApiServiceClient(
		http.DefaultClient,
		baseURL,
		// connect.WithInterceptors(signingInterceptor),
		connect.WithInterceptors(interceptors...),
	)

	return client
}

// NewHTTP2Client creates an HTTP client configured for HTTP/2 cleartext (h2c).
// This is required for ConnectRPC streaming to work properly.
//
// The default Go http.Client uses HTTP/1.1, which causes HTTP 505 errors
// when attempting to use bidirectional streaming with ConnectRPC.
//
// Example usage:
//
//	httpClient := api.NewHTTP2Client()
//	apiClient := apiv1connect.NewApiServiceClient(
//		httpClient,
//		"http://localhost:8080",
//	)
func NewHTTP2Client() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
			MaxHeaderListSize: 10 << 20,
		},
	}
}

// NewHTTP2ClientWithTimeout creates an HTTP/2 client with a custom timeout.
func NewHTTP2ClientWithTimeout(timeout time.Duration) *http.Client {
	client := NewHTTP2Client()
	client.Timeout = timeout
	return client
}
