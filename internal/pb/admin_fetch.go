package pb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/util"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var allowedAdminRoles []string = []string{"ADMIN"}

func (s *AdminServer) GetAllUsers(ctx context.Context, _ *emptypb.Empty) (*m.UserList, error) {
	creds := util.GetFromContext[auth.Claims](ctx, util.JwtKey)
	if !hasRequiredRole(creds.Roles, allowedAdminRoles) {
		return nil, status.Error(
			codes.PermissionDenied,
			"User does not have the authority to query all users",
		)
	}

	users := []*m.User{}
	fetchAll := s.store.Common[db.GetAllUsers]
	rows, err := fetchAll.QueryContext(ctx)
	if err != nil {
		slog.WarnContext(ctx, "Failed to query the list of signed up users")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			uuid    string
			name    string
			email   string
			created time.Time
			updated time.Time
		)
		if err := rows.Scan(&uuid, &name, &email, &created, &updated); err != nil {
			slog.WarnContext(ctx,
				"Failed to query certain user data, response may be incomplete",
				logging.ErrKey, err,
			)
		}
		users = append(users, &m.User{
			Id: &m.UUID{Value: uuid},
			Data: &m.UserData{
				Name:  name,
				Email: email,
			},
			Metadata: &m.UserMetadata{
				CreatedOn: timestamppb.New(created.UTC()),
				UpdatedOn: timestamppb.New(updated.UTC()),
			},
		})
	}

	return &m.UserList{Users: users}, nil
}

func (s *AdminServer) GetUser(ctx context.Context, uuid *m.UUID) (*m.User, error) {
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

func (s *AdminServer) GetUserTasks(context.Context, *m.UUID) (*m.TaskList, error) {
	return nil, nil
}
