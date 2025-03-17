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

func (s *authServer) Login(ctx context.Context, _ *emptypb.Empty) (*m.LoginResponse, error) {
	var (
		name    string
		uuid    string
		created time.Time
		updated time.Time
		hash    string
	)

	creds := util.GetFromContext[auth.Credentials](ctx, util.CredentialsKey)

	stmt := s.store.Common[db.GetUserWithSecret]
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

	if err := auth.ValidateCreds(creds.Secret.String(), hash); err != nil {
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
