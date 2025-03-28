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

func (s *adminServer) GetAllUsers(ctx context.Context, _ *emptypb.Empty) (*m.UserList, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	if err := s.hasAdminRights(ctx, creds); err != nil {
		return nil, err
	}

	allUsers, err := s.db.GetAllUsers(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failure to fetch all users", logging.ErrKey, err)
		return nil, status.Error(codes.Internal, "Failed to fetch all users")
	}

	allUsersPb := make([]*m.User, len(allUsers))

	for i, u := range allUsers {
		allUsersPb[i] = userToPb(u)
	}

	slog.InfoContext(ctx, "success", "user_id", creds.Subject)
	return &m.UserList{
		Users: allUsersPb,
	}, nil
}
