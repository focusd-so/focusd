package api

import (
	"context"

	"connectrpc.com/connect"

	"github.com/focusd-so/focusd/internal/identity"
)

// SigningInterceptor is a client-side interceptor that handles authentication
// by attaching tokens to requests and refreshing them when they expire.
type SigningInterceptor struct {
}

// NewSigningInterceptor creates a new signing interceptor.
// It attempts to load an existing token from the system keychain.
func NewSigningInterceptor() *SigningInterceptor {
	return &SigningInterceptor{}
}

// WrapUnary implements connect.Interceptor for unary RPCs.
func (i *SigningInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		token, err := identity.GetToken(ctx)
		if err != nil {
			return nil, err
		}

		req.Header().Set("Authorization", "Bearer "+token)

		return next(ctx, req)
	}
}

// WrapStreamingClient implements connect.Interceptor for streaming client RPCs.
func (i *SigningInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		token, err := identity.GetToken(ctx)
		if err != nil {
			return nil
		}

		conn := next(ctx, spec)
		conn.RequestHeader().Set("Authorization", "Bearer "+token)

		return conn
	}
}

// WrapStreamingHandler implements connect.Interceptor (no-op for client-side).
func (i *SigningInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
