package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *adminServer) NewUser(ctx context.Context, req *m.UserSignupRequest) (*emptypb.Empty, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	if err := s.hasAdminRights(ctx, creds); err != nil {
		return nil, err
	}

	_, err = s.newUser(ctx, req)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "success", "user_id", creds.UserID)
	return &emptypb.Empty{}, nil
}
