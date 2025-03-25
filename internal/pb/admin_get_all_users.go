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
		roles, err := s.db.GetUserRoles(ctx, u.UserID)
		if err != nil {
			slog.ErrorContext(ctx,
				"Failed to query roles for a specific user",
				"user_id", u.UserID,
				logging.ErrKey, err,
			)
			return nil, status.Errorf(codes.Internal,
				"Failed to query roles for user '%v'", u.UserID,
			)
		}

		allUsersPb[i] = &m.User{
			Id: &m.UUID{Value: u.UserID.String()},
			Data: &m.UserData{
				Name:  u.Name,
				Email: u.Email,
			},
			Metadata: &m.UserMetadata{
				Roles:     roles,
				CreatedOn: timestamppb.New(u.CreatedAt.UTC()),
				UpdatedOn: timestamppb.New(u.UpdatedAt.UTC()),
			},
		}
	}

	slog.InfoContext(ctx, "success", "user_id", creds.UserID)
	return &m.UserList{
		Users: allUsersPb,
	}, nil
}
