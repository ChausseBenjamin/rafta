package pb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"github.com/hashicorp/go-uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *AuthServer) Login(ctx context.Context, creds *m.Credentials) (*m.LoginResponse, error) {
	var (
		name    string
		uuid    string
		created time.Time
		updated time.Time
		hash    string
	)

	stmt := s.store.Common[db.GetSingleUserWithSecret]
	row := stmt.QueryRowContext(ctx, creds.Email)
	err := row.Scan(&name, &uuid, &created, &updated, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.InfoContext(ctx,
				"Failed login attempt, user not found",
				logging.ErrKey, err,
			)
			err = status.Errorf(codes.NotFound,
				"No user with email %s", creds.Email,
			)
			return nil, err
		}
	}

	if err := auth.ValidateCreds(creds.Secret, hash); err != nil {
		slog.InfoContext(ctx,
			"Failed login attempt, invalid credentials",
			logging.ErrKey, err,
		)
		err = status.Errorf(codes.Unauthenticated,
			"Invalid credentials for user %s", creds.Email,
		)
		return nil, err
	}

	roles, err := s.getUserRoles(ctx, uuid)
	if err != nil {
		slog.WarnContext(ctx, "Failed to retrieve user roles for JWT creation")
		return nil, err
	}

	access, refresh, err := s.authMgr.Issue(uuid, roles)
	if err != nil {
		slog.WarnContext(ctx, "Failed to generate new JTW pair")
	}

	slog.InfoContext(ctx, "Successful user login")
	return &m.LoginResponse{
		User: &m.User{
			Id: &m.UUID{Value: uuid},
			Data: &m.UserData{
				Name:  name,
				Email: creds.Email,
			},
			Metadata: &m.UserMetadata{
				CreatedOn: timestamppb.New(created),
				UpdatedOn: timestamppb.New(updated),
			},
		},
		Tokens: &m.JWT{
			Access:  access,
			Refresh: refresh,
		},
	}, nil
}

func (s *AuthServer) Signup(ctx context.Context, info *m.UserCredsRequest) (*m.SignupResponse, error) {
	nbStmt := s.store.Common[db.GetUserCount]
	var userNb int
	err := nbStmt.QueryRowContext(ctx).Scan(&userNb)
	if err != nil {
		slog.WarnContext(ctx, "Failed to query the number of signed up users")
	}

	if !s.cfg.AllowNewUsers || (userNb >= int(s.cfg.MaxUsers)) {
		return nil, status.Errorf(codes.FailedPrecondition, "The server is not accepting new signups at this time")
	}
	// Email
	if _, err := mail.ParseAddress(info.User.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			"Provided email is not properly formatted: '%s'", info.User.Email,
		)
	}
	// Password length
	if l := len(info.UserSecret); l < s.cfg.MinPasswdLen || l > s.cfg.MaxPasswdLen {
		return nil, status.Errorf(codes.InvalidArgument,
			"Provided password is of length %d which is outside of the accepted range [%d-%d]",
			l, s.cfg.MinPasswdLen, s.cfg.MaxPasswdLen,
		)
	}
	// Illegal password characters
	for _, r := range info.UserSecret {
		if r < 32 || r > 126 {
			return nil, status.Errorf(codes.InvalidArgument,
				"Provided password contains illegal characters. Allowed characters are in the [32-126] range (https://www.ascii-code.com)",
			)
		}
	}

	uuid, err := uuid.GenerateUUID()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to generate UUID", logging.ErrKey, err)
		return nil, status.Errorf(codes.Internal,
			"Couldn't generate a unique ID for the new user",
		)
	}

	hash, err := auth.GenerateHash(info.UserSecret)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to hash user password", logging.ErrKey, err)
		return nil, status.Errorf(codes.Internal,
			"Couldn't create a hash for user authentication",
		)
	}

	tx, errTx := s.store.DB.BeginTx(ctx, nil)
	insertUser := tx.StmtContext(ctx, s.store.Common[db.CreateUser])
	insertSecret := tx.StmtContext(ctx, s.store.Common[db.CreateUserSecret])

	_, errInsertUser := insertUser.ExecContext(ctx, uuid, info.User.Name, info.User.Email)
	_, errInsertSecret := insertSecret.ExecContext(ctx, uuid, hash)

	if err := errors.Join(errTx, errInsertUser, errInsertSecret); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, status.Errorf(codes.AlreadyExists,
				"There is already a user with the email: '%s'", info.User.Email,
			)
		}
		slog.ErrorContext(ctx, "Unable to insert user into database", logging.ErrKey, err)
		return nil, status.Errorf(codes.Internal, "Failed to insert new user into the database")
	}

	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx, "Committing user to database failed", logging.ErrKey, err)
		return nil, status.Errorf(codes.Internal, "Failed to insert new user into the database")
	}

	roles, err := s.getUserRoles(ctx, uuid)
	if err != nil {
		slog.WarnContext(ctx, "Failed to retrieve user roles for JWT creation")
		return nil, err
	}

	access, refresh, err := s.authMgr.Issue(uuid, roles)
	if err != nil {
		slog.WarnContext(ctx, "Failed to generate new JTW pair")
	}

	slog.InfoContext(ctx, "Successful user signup")
	return &m.SignupResponse{
		User: &m.User{
			Id: &m.UUID{Value: uuid},
			Data: &m.UserData{
				Name:  info.User.Name,
				Email: info.User.Email,
			},
			Metadata: &m.UserMetadata{
				// NOTE: Since sqlite defaults to the current time
				// we assume the difference with time.Now() is negligible
				// It will be "correctly" sent on next login anyway...
				CreatedOn: timestamppb.Now(),
				UpdatedOn: timestamppb.Now(),
			},
		},
		Tokens: &m.JWT{
			Access:  access,
			Refresh: refresh,
		},
	}, nil
}

// TODO: Refresh(context.Context, *RefreshRequest) (*JWT, error)
