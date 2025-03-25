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
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *authServer) Login(ctx context.Context, _ *emptypb.Empty) (*m.LoginResponse, error) {
	creds, err := auth.GetCreds(ctx, auth.BasicTokenType)
	if err != nil {
		return nil, err
	}

	user, err := s.db.GetUser(ctx, creds.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, status.Error(codes.Internal, "Failed to retrieve user info")
	}

	access, refresh, err := s.auth.Issue(creds.UserID, creds.Roles)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failure during JWT pair generation",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "Failure during JWT generation")
	}

	slog.InfoContext(ctx, "success", "user_id", user.UserID)
	return &m.LoginResponse{
		User: &m.User{
			Id: &m.UUID{Value: user.UserID.String()},
			Data: &m.UserData{
				Name:  user.Name,
				Email: user.Email,
			},
			Metadata: &m.UserMetadata{
				CreatedOn: timestamppb.New(user.CreatedAt.UTC()),
				UpdatedOn: timestamppb.New(user.UpdatedAt.UTC()),
			},
		},
		Tokens: &m.JWT{
			Access:  access,
			Refresh: refresh,
		},
	}, nil
}
