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
)

func (s *adminServer) GetUSer(ctx context.Context, id *m.UUID) (*m.User, error) {
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

	user, err := s.db.GetUser(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to fetch user",
			"user_id", user.UserID,
			logging.ErrKey, err,
		)
		// TODO: differentiate between Internal and NotFound
		return nil, status.Errorf(codes.Internal,
			"Failed to query user '%v'", user.UserID,
		)
	}

	slog.InfoContext(ctx, "success", "user_id", creds.Subject)
	return userToPb(user), nil
}
