package pb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *authServer) Login(ctx context.Context, _ *emptypb.Empty) (*m.LoginResponse, error) {
	creds, err := auth.GetCreds(ctx, auth.BasicTokenType)
	if err != nil {
		return nil, err
	}

	user, err := s.db.GetUser(ctx, creds.Subject)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, status.Error(codes.Internal, "Failed to retrieve user info")
	}

	access, refresh, err := s.auth.Issue(creds.Subject, creds.Roles)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failure during JWT pair generation",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "Failure during JWT generation")
	}

	slog.InfoContext(ctx, "success", "user_id", user.UserID)
	return &m.LoginResponse{
		User: userToPb(user),
		Tokens: &m.JWT{
			Access:  access,
			Refresh: refresh,
		},
	}, nil
}
