package api

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/gen/api/v1/apiv1connect"
)

type authKey struct{}

// authInterceptor implements the connect.Interceptor interface
type authInterceptor struct {
	gormDB *gorm.DB
}

// WrapUnary implements unary RPC authentication
func (i *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// 1. Skip Auth for specific public endpoints (like Handshake)
		if req.Spec().Procedure == apiv1connect.ApiServiceDeviceHandshakeProcedure {
			return next(ctx, req)
		}

		token := req.Header().Get("Authorization")
		// Standard format: "Bearer v2.local.AAAA..."
		token = strings.TrimPrefix(token, "Bearer ")
		token = strings.TrimSpace(token)

		if token == "" {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing token"))
		}

		claims, err := ValidateToken(token)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or expired session"))
		}

		user := User{}
		if err := i.gormDB.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not found"))
		}

		if user.Tier == string(TierFree) {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("user does not have a valid subscription"))
		}

		ctx = context.WithValue(ctx, authKey{}, claims)

		return next(ctx, req)
	}
}

func GetClaims(ctx context.Context) (*UserClaims, error) {
	claims, ok := ctx.Value(authKey{}).(*UserClaims)
	if !ok {
		return nil, errors.New("claims not found")
	}
	return claims, nil
}

func (i *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		// TODO: impl similar to WrapUnary for streaming client when needed
		return next(ctx, conn)
	}
}

// NewAuthInterceptor creates a new authentication interceptor
func NewAuthInterceptor(gormDB *gorm.DB) connect.Interceptor {
	return &authInterceptor{gormDB: gormDB}
}

// GetUser extracts user data from context in your API handlers
func GetUser(ctx context.Context) (*UserClaims, bool) {
	u, ok := ctx.Value(authKey{}).(*UserClaims)
	return u, ok
}
