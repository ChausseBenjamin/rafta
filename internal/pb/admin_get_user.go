package pb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *adminServer) GetUser(ctx context.Context, uuid *m.UUID) (*m.User, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		return nil, err
	}

	if !hasRequiredRole(creds.Roles, allowedAdminRoles) {
		return nil, status.Error(
			codes.PermissionDenied,
			"User does not have the authority to query all users",
		)
	}

	var (
		name    string
		email   string
		created time.Time
		updated time.Time
	)

	fetch := s.store.Common[db.GetUser]
	row := fetch.QueryRowContext(ctx, uuid.Value)
	if err := row.Scan(&name, &email, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(ctx, "Admin did not find a user he was querying", "user", uuid.Value)
			return nil, status.Errorf(codes.NotFound,
				"No user with userID '%s' was found within the database", uuid.Value,
			)
		} else {
			slog.ErrorContext(ctx, "Failed to query db for a specific user",
				logging.ErrKey, err,
			)
			return nil, status.Errorf(codes.Internal, "Failed to query specific user data")
		}
	}

	return &m.User{
		Id: &m.UUID{Value: uuid.Value},
		Data: &m.UserData{
			Name:  name,
			Email: email,
		},
		Metadata: &m.UserMetadata{
			CreatedOn: timestamppb.New(created),
			UpdatedOn: timestamppb.New(updated),
		},
	}, nil
}
