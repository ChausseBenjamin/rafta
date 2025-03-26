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
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *raftaServer) GetUserInfo(ctx context.Context, _ *emptypb.Empty) (*m.User, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	user, err := s.db.GetUser(ctx, creds.UserID)
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

	roles, err := s.db.GetUserRoles(ctx, user.UserID)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to query roles for the user",
			"user_id", user.UserID,
			logging.ErrKey, err,
		)
		return nil, status.Errorf(codes.Internal,
			"Failed to query roles for user '%v'", user.UserID,
		)
	}

	slog.InfoContext(ctx, "success", "user_id", creds.UserID)
	return &m.User{
		Id: &m.UUID{Value: creds.UserID.String()},
		Data: &m.UserData{
			Name:  user.Name,
			Email: user.Email,
		},
		Metadata: &m.UserMetadata{
			Roles:     roles,
			CreatedOn: timestamppb.New(user.CreatedAt.UTC()),
			UpdatedOn: timestamppb.New(user.UpdatedAt.UTC()),
		},
	}, nil
}
