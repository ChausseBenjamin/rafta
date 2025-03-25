package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const msgNoNewUser = "The server is not accepting new users at this time. If you are and administrator and wish to circumvent this, please use the 'Admin/CreateUser' endpoint"

func (s *authServer) Signup(ctx context.Context, req *m.UserSignupRequest) (*m.SignupResponse, error) {
	if !s.cfg.AllowNewUsers {
		slog.WarnContext(ctx,
			"Blocked signup attempt as public signups are currently closed",
		)
		return nil, status.Error(codes.FailedPrecondition, msgNoNewUser)
	}

	// 0 implies no user limit
	if s.cfg.MaxUsers > 0 {
		userCount, err := s.db.GetUserCount(ctx)
		if err != nil {
			slog.ErrorContext(ctx,
				"Failed to determine the platforms user count to limit signups",
				logging.ErrKey, err,
			)
			return nil, status.Error(codes.Internal,
				"Failed to determine if the platform accepts new users",
			)
		}

		if userCount >= int64(s.cfg.MaxUsers) {
			slog.WarnContext(ctx,
				"Blocked signup attempt as server has reached max-user capacity",
			)
			return nil, status.Error(codes.FailedPrecondition, msgNoNewUser)
		}
	}

	user, err := s.newUser(ctx, req)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(user.Id.Value)
	if err != nil {
		slog.ErrorContext(ctx,
			"failed to parse user ID during signup",
			"user_id", user.Id.Value,
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"Failed to create a user identifier",
		)
	}

	acess, refresh, err := s.auth.Issue(userID, user.Metadata.Roles)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to generate JWT pair",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "Failure during token generation")
	}

	slog.InfoContext(ctx, "success", "user_id", user.Id)
	return &m.SignupResponse{
		User: user,
		Tokens: &m.JWT{
			Access:  acess,
			Refresh: refresh,
		},
	}, nil
}
