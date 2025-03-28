package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/util"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *adminServer) DeleteUser(ctx context.Context, id *m.UUID) (*emptypb.Empty, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	if err := s.hasAdminRights(ctx, creds); err != nil {
		return nil, err
	}

	userID, err := util.ParseUUID(ctx, util.ParseUUIDParams{
		Str: id.Value, Subject: "user_id",
		Critical: true, Implication: codes.InvalidArgument,
	})
	if err != nil {
		return nil, err
	}

	if err := s.db.DeleteUser(ctx, userID); err != nil {
		slog.ErrorContext(ctx, "Failure during user deletion", logging.ErrKey, err)
		return nil, status.Error(codes.Internal, "Failed to delete user")
	}

	slog.InfoContext(ctx, "success", "user_id", creds.Subject)
	return &emptypb.Empty{}, nil
}
