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
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *adminServer) GetUSer(ctx context.Context, id *m.UUID) (*m.User, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	if err := s.hasAdminRights(ctx, creds); err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(id.Value)
	if err != nil {
		slog.WarnContext(ctx,
			"failed to parse provided userID",
			"user_id", id.Value,
			logging.ErrKey, err,
		)
		return nil, status.Errorf(codes.InvalidArgument,
			"Failed to parse provided user id. Parser returned '%v'", err,
		)
	}

	user, err := s.db.GetUser(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to fetch user",
			"user_id", user.UserID,
			logging.ErrKey, err,
		)
		return nil, status.Errorf(codes.Internal,
			"Failed to query user '%v'", user.UserID,
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
		Id: &m.UUID{Value: userID.String()},
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
