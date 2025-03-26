package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *raftaServer) DeleteUser(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	if err := s.db.DeleteUser(ctx, creds.UserID); err != nil {
		slog.ErrorContext(ctx, "failure while trying to delete user",
			"user_id", creds.UserID,
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"a failure occured while trying to close account",
		)
	}

	return &emptypb.Empty{}, nil
}
