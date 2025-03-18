package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *raftaServer) DeleteUser(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		return nil, err
	}
	cmd := s.store.Common[db.DeleteUser]
	resp, err := cmd.ExecContext(ctx, creds.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "User request to close account failed",
			logging.ErrKey, err,
			db.RespMsgKey, resp,
		)
		return nil, status.Errorf(codes.Internal,
			"An error occured wile closing the account",
		)
	}
	if i, err := resp.RowsAffected(); i == 0 && err == nil {
		return nil, status.Errorf(codes.NotFound,
			"User %s does not exist in the database", creds.UserID,
		)
	}
	slog.InfoContext(ctx, "deleted user", db.RespMsgKey, resp)
	return &emptypb.Empty{}, nil
}
