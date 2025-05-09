// server_tools.go contains logic bits that are used by multiple endpoints
// to minimize code duplication (ex: both `Admin/CreateUser` and
// `Auth/Signup` need the same code to instantiate a new user).
package pb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/mail"
	"slices"
	"strings"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/database"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/sec"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *protoServer) newUser(ctx context.Context, req *m.UserSignupRequest) (*m.User, error) {
	if err := validateEmail(ctx, req.User.Email); err != nil {
		return nil, err
	}

	emailExists, err := s.db.UserWithEmailExists(ctx, req.User.Email)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to assert the signup email is unique",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "Failure while checking if email is taken")
	}
	if emailExists {
		slog.WarnContext(ctx,
			"Attempt to reuse existing email blocked",
			"email", req.User.Email,
		)
		return nil, status.Errorf(codes.AlreadyExists,
			"A user with the email '%s' is already registered", req.User.Email,
		)
	}

	if err := s.auth.ValidatePasswd(req.UserSecret); err != nil {
		return nil, err
	}

	hash, salt, err := sec.GenerateHash(req.UserSecret)
	if err != nil {
		slog.ErrorContext(ctx, "Failure to hash user password", logging.ErrKey, err)
		return nil, status.Errorf(codes.Internal,
			"Couldn't create a hash for user authentication",
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx,
			"Signup transaction initialization failure",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "Failed to begin user creation")
	}
	defer tx.Rollback()
	db := s.db.WithTx(tx)

	user, err := db.NewUser(ctx, database.NewUserParams{
		Name:  req.User.Name,
		Email: req.User.Email,
	})
	if err != nil {
		slog.ErrorContext(ctx, "User creation failure", logging.ErrKey, err)
		return nil, status.Error(codes.Internal, "Failed to create new user")
	}

	if err := db.NewUserSecret(ctx, database.NewUserSecretParams{
		UserID: user.UserID,
		Salt:   salt,
		Hash:   hash,
	}); err != nil {
		slog.ErrorContext(ctx, "User credentials insertion failure", logging.ErrKey, err)
		return nil, status.Error(codes.Internal, "Failed to store user credentials hash")
	}

	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx,
			"Failure to commit user creation transaction",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "Failed to complete user creation")
	}

	return userToPb(user), nil
}

func (s *adminServer) hasAdminRights(ctx context.Context, creds *auth.Credendials) error {
	isAuthorized := false
	for _, acceptedRole := range adminRoles {
		if slices.Contains(creds.Roles, acceptedRole) {
			isAuthorized = true
			break
		}
	}
	if !isAuthorized {
		slog.WarnContext(ctx, "Unauthorized user attempted to create a user")
		return status.Error(codes.PermissionDenied, "Insuficient privileges to perform operation")
	}
	return nil
}

func (s *protoServer) updateUser(ctx context.Context, userID uuid.UUID, data *m.UserData) (*timestamppb.Timestamp, error) {
	if err := validateEmail(ctx, data.Email); err != nil {
		return nil, err
	}

	updated, err := s.db.UpdateUser(ctx, database.UpdateUserParams{
		UserID: userID,
		Name:   data.Name,
		Email:  data.Email,
	})
	if err != nil {
		log := slog.With(logging.ErrKey, err)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			log.WarnContext(ctx,
				"No user with the provided userID exists",
				"user_id", userID,
			)
			return nil, status.Errorf(codes.NotFound,
				"user not found: '%v'", userID,
			)
		case strings.Contains(err.Error(), "constraint"):
			log.WarnContext(ctx,
				"A user with this email already exists",
				"email", data.Email,
			)
			return nil, status.Errorf(codes.AlreadyExists,
				"email already in use: '%v'", data.Email,
			)
		default:
			log.ErrorContext(ctx, "Database error")
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}
	return timestamppb.New(updated.UTC()), nil
}

func (s *protoServer) updateUserCredentials(
	ctx context.Context,
	userID uuid.UUID,
	passwd string,
) (*timestamppb.Timestamp, error) {
	if err := s.auth.ValidatePasswd(passwd); err != nil {
		return nil, err
	}

	hash, salt, err := sec.GenerateHash(passwd)
	if err != nil {
		slog.ErrorContext(ctx, "Failure to hash user password", logging.ErrKey, err)
		return nil, status.Errorf(codes.Internal,
			"Couldn't create a hash for user authentication",
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to begin transaction",
			"user_id", userID,
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"failed to start credentials update",
		)
	}
	defer tx.Rollback()
	db := s.db.WithTx(tx)

	if err := db.UpdateUserSecret(ctx, database.UpdateUserSecretParams{
		UserID: userID,
		Salt:   salt,
		Hash:   hash,
	}); err != nil {
		slog.ErrorContext(ctx, "failure during credentials update",
			"user_id", userID,
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "failed to update credentials")
	}

	modified, err := db.UpdateUserModified(ctx)
	if err != nil {
		slog.ErrorContext(ctx,
			"failed to update 'last_modified' state for user",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"failed to update user metadata. aborting",
		)
	}

	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx,
			"failure to commit credentials update transaction",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "failed to complete credentials update")
	}

	return timestamppb.New(modified.UTC()), nil
}

func validateEmail(ctx context.Context, email string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		slog.WarnContext(ctx, "provided email is not valid",
			logging.ErrKey, err,
		)
		return status.Error(codes.InvalidArgument,
			"failed to parse provided email. Ensure it is in a valid format",
		)
	}
	return nil
}

func userToPb(u database.User) *m.User {
	return &m.User{
		Id: &m.UUID{Value: u.UserID.String()},
		Data: &m.UserData{
			Name:  u.Name,
			Email: u.Email,
		},
		Metadata: &m.UserMetadata{
			CreatedOn: timestamppb.New(u.CreatedOn.UTC()),
			UpdatedOn: timestamppb.New(u.UpdatedOn.UTC()),
		},
	}
}
