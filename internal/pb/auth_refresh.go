package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/util"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

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
