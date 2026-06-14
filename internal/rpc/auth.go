package rpc

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/pocketbase/pocketbase/core"
)

type ctxKey string

const userIDKey ctxKey = "cortex_user_id"

// userID returns the authenticated user id from context, or "" in local
// (unauthenticated) mode.
func userID(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

// AuthInterceptor reads the Authorization bearer token from request metadata,
// validates it against PocketBase, and stores the resulting user id in context.
//
// When no valid token is present the request proceeds in local single-user
// mode (no owner scoping). This is intended for the local desktop daemon where
// the only client is the Wails app on loopback. For multi-user/remote
// deployments, enforce auth by rejecting requests with an empty user id.
func AuthInterceptor(app core.App) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token := strings.TrimPrefix(req.Header().Get("Authorization"), "Bearer ")
			token = strings.TrimSpace(token)
			if token != "" {
				if rec, err := app.FindAuthRecordByToken(token, core.TokenTypeAuth); err == nil && rec != nil {
					ctx = context.WithValue(ctx, userIDKey, rec.Id)
				}
			}
			return next(ctx, req)
		}
	}
}
