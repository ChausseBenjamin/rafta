package pb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/mail"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *AdminServer) DeleteUser(ctx context.Context, id *m.UUID) (*emptypb.Empty, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		return nil, err
	}
	if !hasRequiredRole(creds.Roles, allowedAdminRoles) {
		return nil, status.Error(
			codes.PermissionDenied,
			"User does not have the authority to delete another users",
		)
	}

	cmd := s.store.Common[db.DeleteUser]
	resp, err := cmd.ExecContext(ctx, id.Value)
	if err != nil {
		slog.ErrorContext(ctx, "Admin request to delete user failed",
			logging.ErrKey, err,
			db.RespMsgKey, resp,
		)
		return nil, err
	}
	if i, err := resp.RowsAffected(); i == 0 && err == nil {
		return nil, status.Errorf(codes.NotFound,
			"User %s does not exist in the database", id.Value,
		)
	}
	slog.InfoContext(ctx, "deleted user", db.RespMsgKey, resp)
	return &emptypb.Empty{}, nil
}

func (s *AdminServer) UpdateUser(ctx context.Context, user *m.User) (*emptypb.Empty, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		return nil, err
	}
	if !hasRequiredRole(creds.Roles, allowedAdminRoles) {
		return nil, status.Error(
			codes.PermissionDenied,
			"User does not have the authority to update another user",
		)
	}

	// Ensure the user about to get updated actually exists
	assertExistence := s.store.Common[db.AssertUserExists]
	row := assertExistence.QueryRowContext(ctx, user.Id.Value)
	userExists := false
	if err := row.Scan(&userExists); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.ErrorContext(ctx,
				"Failed to query the database for a given userID",
				"user", user.Id.Value,
				logging.ErrKey, err,
			)
			return nil, status.Error(codes.Internal,
				"An error occurred while searching for the user to update",
			)
		}
		slog.WarnContext(ctx,
			"An attempt to update a nonexistent user was made",
			"user", user.Id.Value,
		)
		return nil, status.Errorf(codes.NotFound,
			"User %s does not exist", user.Id.Value,
		)
	}

	// Ensure an admin doesn't mistakenly give two users the same email:
	checkEmailCollision := s.store.Common[db.GetUserIDFromEmail]
	rows, err := checkEmailCollision.QueryContext(ctx, user.Data.Email)
	if err != nil {
		slog.WarnContext(ctx, "Failed to confirm uniqueness of email while updating the user")
		return nil, status.Error(codes.Internal, "Failed to confirm uniqueness of email")
	}
	defer rows.Close()

	// There should be only 1 possible uuid per email. For-loop is used for safety
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			slog.WarnContext(ctx, "Failed to confirm uniqueness of email while updating the user")
			return nil, status.Error(codes.Internal, "Failed to confirm uniqueness of email")
		}
		if email != "" && email != user.Id.Value {
			slog.WarnContext(ctx,
				"Admin attempted to update a user with an email which already exists",
			)
			return nil, status.Error(codes.FailedPrecondition,
				"Cannot update a user with an email that already exists in the system",
			)
		}
	}

	// Ensure the new email is still a valid email
	if _, err := mail.ParseAddress(user.Data.Email); err != nil {
		slog.WarnContext(ctx,
			"Admin attempted to update a user with an invalid email format",
			"email", user.Data.Email,
		)
		return nil, status.Errorf(codes.InvalidArgument,
			"Cannot update user with an invalid email: '%s'", user.Data.Email,
		)
	}

	stmt := s.store.Common[db.UpdateUser]
	_, err = stmt.ExecContext(ctx,
		user.Data.Name,
		user.Data.Email,
		time.Now().UTC(),
		user.Id.Value,
	)
	if err != nil {
		slog.ErrorContext(ctx,
			"There was a failure attempting to update a user",
			logging.ErrKey, err,
		)
		return nil, status.Errorf(codes.Internal,
			"Failed to update the desired user: %s", user.Id.Value,
		)
	}

	return &emptypb.Empty{}, nil
}
