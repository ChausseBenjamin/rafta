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

func (s *adminServer) UpdateCredentials(ctx context.Context, req *m.ChangePasswdRequest) (*emptypb.Empty, error) {
	if err := s.validatePasswd(req.Secret); err != nil {
		return nil, err
	}

	hash, err := auth.GenerateHash(req.Secret)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to hash user password", logging.ErrKey, err)
		return nil, status.Errorf(codes.Internal,
			"Couldn't create a hash for user authentication",
		)
	}

	stmt := s.store.Common[db.UpdateUserPasswd]
	_, err = stmt.ExecContext(ctx, hash, req.Id.Value)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to change user password",
			"uuid", req.Id.Value,
			logging.ErrKey, err,
		)
		return nil, status.Errorf(codes.Internal,
			"Failed to update password for user '%s'", req.Id.Value,
		)
	}

	return &emptypb.Empty{}, nil
}
