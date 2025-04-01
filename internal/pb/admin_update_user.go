package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/util"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *adminServer) UpdateUser(ctx context.Context, user *m.User) (*emptypb.Empty, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	if err := s.hasAdminRights(ctx, creds); err != nil {
		return nil, err
	}

	userID, err := util.ParseUUID(ctx, util.ParseUUIDParams{
		Str: user.Id.Value, Subject: "user_id",
		Critical: true, Implication: codes.InvalidArgument,
	})
	if err != nil {
		return nil, err
	}

	_, err = s.updateUser(ctx, userID, user.Data)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "success", "user_id", creds.Subject)
	return &emptypb.Empty{}, nil
}
