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

func (s *authServer) Signup(ctx context.Context, req *m.UserSignupRequest) (*m.SignupResponse, error) {
	nbStmt := s.store.Common[db.GetUserCount]
	var userCount int
	err := nbStmt.QueryRowContext(ctx).Scan(&userCount)
	if err != nil {
		slog.WarnContext(ctx, "Failed to query the number of signed up users")
	}

	if !s.cfg.AllowNewUsers || (userCount >= int(s.cfg.MaxUsers)) {
		return nil, status.Errorf(codes.FailedPrecondition, "The server is not accepting new signups at this time")
	}

	user, err := s.newUser(ctx, req)
	if err != nil {
		return nil, err
	}

	roles, err := s.getUserRoles(ctx, user.Id.Value)
	if err != nil {
		slog.WarnContext(ctx, "Failed to retrieve user roles for JWT creation")
		return nil, err
	}

	access, refresh, err := s.authMgr.Issue(user.Id.Value, roles)
	if err != nil {
		slog.WarnContext(ctx, "Failed to generate new JTW pair")
	}

	slog.InfoContext(ctx, "Successful user signup")
	return &m.SignupResponse{
		User: &m.User{
			Id: &m.UUID{Value: user.Id.Value},
			Data: &m.UserData{
				Name:  req.User.Name,
				Email: req.User.Email,
			},
			Metadata: &m.UserMetadata{
				// NOTE: Since sqlite defaults to the current time
				// we assume the difference with time.Now() is negligible
				// It will be "correctly" sent on next login anyway...
				CreatedOn: timestamppb.New(time.Now().UTC()),
				UpdatedOn: timestamppb.New(time.Now().UTC()),
			},
		},
		Tokens: &m.JWT{
			Access:  access,
			Refresh: refresh,
		},
	}, nil
}

func (a *authServer) Refresh(ctx context.Context, _ *emptypb.Empty) (*m.JWT, error) {
	claims := util.GetFromContext[auth.Claims](ctx, util.JwtKey)
	if claims == nil {
		slog.ErrorContext(ctx,
			"Reached the Refresh endpoint without a valid token",
		)
		return nil, status.Error(codes.Internal, "Reached the Refresh endpoint without a valid token")
	}

	if t := claims.Type; t != "refresh" {
		slog.WarnContext(ctx, "Client attempt to refresh with a non-refresh token", "provided", t)
	}

	// Ensure the refresh token cannot be reused twice by blacklisting it
	stmt := a.store.Common[db.RevokeToken]
	_, err := stmt.ExecContext(ctx, claims.ID, claims.ExpiresAt.Time.UTC())
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to enforce single-use of the refresh token",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"Failure to enforce single-use policy for refresh token.",
		)
	}

	access, refresh, err := a.authMgr.Issue(claims.UserID, claims.Roles)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to issue new JWT token pair (access+refresh).",
		)
		return nil, status.Error(codes.Internal, "JWT token generation failed")
	}

	return &m.JWT{
		Access:  access,
		Refresh: refresh,
	}, nil
}
