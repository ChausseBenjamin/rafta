package pb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *adminServer) checkUserExistence(ctx context.Context, userID, action string) error {
	assertExistence := s.store.Common[db.AssertUserExists]
	row := assertExistence.QueryRowContext(ctx, userID)
	userExists := false
	if err := row.Scan(&userExists); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.ErrorContext(ctx,
				"Failed to query the database for a given userID",
				"user", userID,
				logging.ErrKey, err,
			)
			return status.Error(codes.Internal,
				"An error occurred while searching for the user to update",
			)
		}
		slog.WarnContext(ctx,
			fmt.Sprintf("An attempt to %s a nonexistent user was made", action),
			"user", userID,
		)
		return status.Errorf(codes.NotFound,
			"User %s does not exist", userID,
		)
	}
	return nil
}

func (s *adminServer) checkEmailCollision(ctx context.Context, email, userID, action string) error {
	checkEmailCollision := s.store.Common[db.GetUserIDFromEmail]
	rows, err := checkEmailCollision.QueryContext(ctx, email)
	if err != nil {
		slog.WarnContext(ctx, "Failed to confirm uniqueness of email while updating the user")
		return status.Error(codes.Internal, "Failed to confirm uniqueness of email")
	}
	defer rows.Close()

	// There should be only 1 possible uuid per email. For-loop is used for safety
	for rows.Next() {
		var existingUserID string
		if err := rows.Scan(&existingUserID); err != nil {
			slog.WarnContext(ctx, "Failed to confirm uniqueness of email while updating the user")
			return status.Error(codes.Internal, "Failed to confirm uniqueness of email")
		}
		if existingUserID != "" && existingUserID != userID {
			slog.WarnContext(ctx,
				fmt.Sprintf("Admin attempted to %s a user with an email which already exists", action),
			)
			return status.Error(codes.FailedPrecondition,
				"Cannot update a user with an email that already exists in the system",
			)
		}
	}
	return nil
}
