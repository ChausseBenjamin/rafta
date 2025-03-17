package pb

import (
	"context"
	"log/slog"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/db"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
