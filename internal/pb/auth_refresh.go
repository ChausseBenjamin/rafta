package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/database"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *authServer) Refresh(ctx context.Context, _ *emptypb.Empty) (*m.JWT, error) {
	creds, err := auth.GetCreds(ctx, auth.RefreshTokenType)
	if err != nil {
		return nil, err
	}

	tokenID, err := uuid.Parse(creds.ID)
	if err != nil {
		slog.ErrorContext(ctx,
			"failed to parse refresh token id",
			"token", tokenID,
			logging.ErrKey, err,
		)
		// Internal error since token was successfully parse in the interceptor
		return nil, status.Error(codes.Internal,
			"Failed to parse refresh token id",
		)
	}

	err = s.db.RevokeToken(ctx, database.RevokeTokenParams{
		TokenID: tokenID,
		Expiry:  creds.ExpiresAt.Time.UTC(),
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to add revoked token to database")
		return nil, status.Error(codes.Internal,
			"failed to revoke refresh token. Operation aborted",
		)
	}
	if err := s.db.CleanupExpiredToken(ctx, tokenID); err != nil {
		// Non-critical, clients shouldn't be blocked by housekeeping errors
		slog.WarnContext(ctx, "Failed to schedule token deletion after expiry",
			logging.ErrKey, err,
		)
	}

	// No need to fetch the database, roles are already provided
	access, refresh, err := s.auth.Issue(creds.UserID, creds.Roles)
	if err != nil {
		slog.ErrorContext(ctx,
			"failed to issue a JWT pair",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "JWT generation failed")
	}

	slog.InfoContext(ctx, "success", "user_id", creds.UserID)
	return &m.JWT{
		Access:  access,
		Refresh: refresh,
	}, nil
}
