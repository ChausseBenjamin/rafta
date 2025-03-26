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

func (s *adminServer) UpdateCredentials(ctx context.Context, req *m.ChangePasswdRequest) (*emptypb.Empty, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	if err := s.hasAdminRights(ctx, creds); err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(req.Id.Value)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse token ID", logging.ErrKey, err)
		return nil, status.Error(codes.InvalidArgument, "Invalid target user ID")
	}

	if _, err := s.updateUserCredentials(ctx, userID, req.Secret); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "success", "user_id", creds.UserID)
	return &emptypb.Empty{}, nil
}
