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

func (s *adminServer) DeleteUser(ctx context.Context, id *m.UUID) (*emptypb.Empty, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	if err := s.hasAdminRights(ctx, creds); err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(id.Value)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse user ID", logging.ErrKey, err)
		return nil, status.Error(codes.InvalidArgument, "Invalid target user ID")
	}

	if err := s.db.DeleteUser(ctx, userID); err != nil {
		slog.ErrorContext(ctx, "Failure during user deletion", logging.ErrKey, err)
		return nil, status.Error(codes.Internal, "Failed to delete user")
	}

	slog.InfoContext(ctx, "success", "user_id", creds.UserID)
	return &emptypb.Empty{}, nil
}
