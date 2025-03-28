package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *raftaServer) GetUserInfo(ctx context.Context, _ *emptypb.Empty) (*m.User, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	user, err := s.db.GetUser(ctx, creds.Subject)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to fetch user info",
			"user_id", user.UserID,
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"Failed to query user info",
		)
	}

	slog.InfoContext(ctx, "success", "user_id", creds.Subject)
	return userToPb(user), nil
}
