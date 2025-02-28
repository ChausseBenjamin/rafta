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
			Id: &m.UUID{
				Value: uuid,
			},
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

func (s *AuthServer) getUserRoles(ctx context.Context, userID string) ([]string, error) {
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
