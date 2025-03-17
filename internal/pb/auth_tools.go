package pb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"slices"
	"strings"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/util"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"github.com/hashicorp/go-uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxUUIDGenAttempts = 5

// getUserRoles is meant to be used when creating JWT tokens.
// Since login and signup only provide an email as an identifier, role
// information has to be fetched from the database.
func (s *authServer) getUserRoles(ctx context.Context, userID string) ([]string, error) {
	stmt := s.store.Common[db.GetUserRoles]
	rows, err := stmt.QueryContext(ctx, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

func validatePasswd(p string, minLen int, maxLen int) error {
	// Password length
	if l := len(p); l < minLen || l > maxLen {
		return status.Errorf(codes.InvalidArgument,
			"Provided password is of length %d which is outside of the accepted range [%d-%d]",
			l, minLen, maxLen,
		)
	}
	// Illegal password characters
	for _, r := range p {
		if r < 32 || r > 126 {
			return status.Errorf(codes.InvalidArgument,
				"Provided password contains illegal characters. Allowed characters are in the [32-126] range (https://www.ascii-code.com)",
			)
		}
	}
	return nil
}

// Since there are some encpoints that don't require authentication (ex: Signup)
// The JWT interceptor can let unauthenticated request pass through. Not catching this
// can (and will) lead to nil pointer dereferences.
func getCreds(ctx context.Context) (*auth.Claims, error) {
	creds := util.GetFromContext[auth.Claims](ctx, util.JwtKey)
	if creds == nil {
		slog.WarnContext(ctx, "User is not authenticated, cannot proceed with request")
		return nil,
			status.Error(codes.Unauthenticated, "Current endpoint requires JWT authentication to proceed, cannot continue")
	} else {
		return creds, nil
	}
}

func hasRequiredRole(claimedRoles []string, allowedRoles []string) bool {
	for _, role := range claimedRoles {
		if slices.Contains(allowedRoles, role) {
			return true
		}
	}
	return false
}

// newUser is a centralized function for signing up so that admins creating users or public clients
// signing go through the same logic flow.
func (s *protoServer) newUser(ctx context.Context, req *m.UserSignupRequest) (*m.User, error) {
	// Email
	if _, err := mail.ParseAddress(req.User.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			"Provided email is not properly formatted: '%s'", req.User.Email,
		)
	}

	if err := validatePasswd(req.UserSecret, s.cfg.MinPasswdLen, s.cfg.MaxPasswdLen); err != nil {
		return nil, err
	}

	var (
		exists   bool = true
		attempts int
		id       string
		err      error
	)
	uniqueCheck := s.store.Common[db.AssertUserExists]
	for !exists {
		attempts++
		id, err = uuid.GenerateUUID()
		if err != nil {
			slog.ErrorContext(ctx, "Failed to generate UUID", logging.ErrKey, err)
			return nil, status.Errorf(codes.Internal,
				"Couldn't generate a unique ID for the new user",
			)
		}
		err := uniqueCheck.QueryRowContext(ctx, id).Scan(&exists)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to ensure the generated UUID was unique",
				"uuid", id,
				logging.ErrKey, err,
			)
			return nil, status.Error(codes.Internal,
				"Failed to generate a valid UUID",
			)
		}
		if attempts >= maxUUIDGenAttempts {
			slog.ErrorContext(ctx,
				"Max uuid generation attemps reached during signup",
				"attempts", maxUUIDGenAttempts,
			)
			return nil, status.Errorf(codes.Internal,
				"Max uuid generation attemps reached",
			)
		}

	}

	hash, err := auth.GenerateHash(req.UserSecret)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to hash user password", logging.ErrKey, err)
		return nil, status.Errorf(codes.Internal,
			"Couldn't create a hash for user authentication",
		)
	}

	tx, errTx := s.store.DB.BeginTx(ctx, nil)
	insertUser := tx.StmtContext(ctx, s.store.Common[db.CreateUser])
	insertSecret := tx.StmtContext(ctx, s.store.Common[db.CreateUserSecret])

	_, errInsertUser := insertUser.ExecContext(ctx, id, req.User.Name, req.User.Email)
	_, errInsertSecret := insertSecret.ExecContext(ctx, id, hash)

	if err := errors.Join(errTx, errInsertUser, errInsertSecret); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, status.Errorf(codes.AlreadyExists,
				"There is already a user with the email: '%s'", req.User.Email,
			)
		}
		slog.ErrorContext(ctx, "Unable to insert user into database", logging.ErrKey, err)
		return nil, status.Errorf(codes.Internal, "Failed to insert new user into the database")
	}

	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx, "Committing user to database failed", logging.ErrKey, err)
		return nil, status.Errorf(codes.Internal, "Failed to insert new user into the database")
	}
	return &m.User{
		Id:   &m.UUID{Value: id},
		Data: req.User,
	}, nil
}

// checkUserExistence helps preventing errors if an admin tries to update/delete a non-existent user
func (s *protoServer) checkUserExistence(ctx context.Context, userID, action string) error {
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

// checkEmailCollision helps avoid two users getting assigned the same email/username on the server
func (s *protoServer) checkEmailCollision(ctx context.Context, email, userID, action string) error {
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
