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
	m "github.com/ChausseBenjamin/rafta/pkg/model"
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
			slog.InfoContext(ctx, "Failed login attempt, user not found", logging.ErrKey, err)
			err = status.Errorf(codes.NotFound,
				"No user with email %s", creds.Email,
			)
			return nil, err
		}
	}

	if err := auth.ValidateCreds(creds.Secret, hash); err != nil {
		slog.InfoContext(ctx, "Failed login attempt, invalid credentials", logging.ErrKey, err)
		err = status.Errorf(codes.Unauthenticated,
			"Invalid credentials for user %s", creds.Email,
		)
		return nil, err
	}

	slog.InfoContext(ctx, "Successful user login")
	return &m.LoginResponse{
		User: &m.User{
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
			// TODO: Generate a JWT access and refresh token
		},
	}, nil
}
