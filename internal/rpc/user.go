package rpc

import (
	"context"

	"connectrpc.com/connect"
	"github.com/NexVed/Cortex/gen/cortex/v1"
	"github.com/NexVed/Cortex/gen/cortex/v1/cortexv1connect"
	"github.com/NexVed/Cortex/internal/db"
)

// AuthServiceHandler implements the ConnectRPC AuthService.
type AuthServiceHandler struct {
	*Handlers
}

var _ cortexv1connect.AuthServiceHandler = (*AuthServiceHandler)(nil)

func (h *AuthServiceHandler) GetCurrentUser(
	ctx context.Context,
	req *connect.Request[cortexv1.GetCurrentUserRequest],
) (*connect.Response[cortexv1.GetCurrentUserResponse], error) {
	uid := userID(ctx)
	if uid == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	user, err := h.App.FindRecordById(db.CollUsers, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&cortexv1.GetCurrentUserResponse{
		Id:              user.Id,
		DisplayName:     user.GetString("display_name"),
		GithubUsername:  user.GetString("github_username"),
		GithubAvatarUrl: user.GetString("github_avatar_url"),
		Email:           user.GetString("email"),
	}), nil
}
