package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *raftaServer) UpdateCredentials(ctx context.Context, psswd *m.PasswdMessage) (*emptypb.Empty, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to assert identity passed authentication",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"Could not establish identity after authentication",
		)
	}

	if err := s.validatePasswd(psswd.Secret); err != nil {
		return nil, err
	}

	hash, err := auth.GenerateHash(psswd.Secret)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to hash user password", logging.ErrKey, err)
		return nil, status.Errorf(codes.Internal,
			"Couldn't create a hash for user authentication",
		)
	}

	stmt := s.store.Common[db.UpdateUserPasswd]
	_, err = stmt.ExecContext(ctx, hash, creds.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to change user password",
			"uuid", creds.UserID,
			logging.ErrKey, err,
		)
		return nil, status.Errorf(codes.Internal,
			"Failed to update password for user '%s'", creds.UserID,
		)
	}

	return &emptypb.Empty{}, nil
}
