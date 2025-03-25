package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *adminServer) UpdateUser(ctx context.Context, user *m.User) (*emptypb.Empty, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	if err := s.hasAdminRights(ctx, creds); err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(user.Id.Value)
	if err != nil {
		slog.ErrorContext(ctx,
			"failed to parse provided user ID",
			"user_id", user.Id.Value,
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"Failed to create a user identifier",
		)
	}

	_, err = s.updateUser(ctx, userID, user.Data)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "success", "user_id", creds.UserID)
	return &emptypb.Empty{}, nil
}
