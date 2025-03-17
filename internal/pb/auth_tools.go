package pb

import (
	"context"
	"log/slog"
	"slices"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// getUserRoles is meant to be used when creating JWT tokens.
// Since login and signup only provide an email as an identifier, role
// information has to be fetched from the database.
func (s *authServer) getUserRoles(ctx context.Context, userID string) ([]string, error) {
	stmt := s.store.Common[db.GetUserRoles]
	rows, err := stmt.QueryContext(ctx, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

// Since there are some encpoints that don't require authentication (ex: Signup)
// The JWT interceptor can let unauthenticated request pass through. Not catching this
// can (and will) lead to nil pointer dereferences.
func getCreds(ctx context.Context) (*auth.Claims, error) {
	creds := util.GetFromContext[auth.Claims](ctx, util.JwtKey)
	if creds == nil {
		slog.WarnContext(ctx, "User is not authenticated, cannot proceed with request")
		return nil,
			status.Error(codes.Unauthenticated, "Current endpoint requires JWT authentication to proceed, cannot continue")
	} else {
		return creds, nil
	}
}

func hasRequiredRole(claimedRoles []string, allowedRoles []string) bool {
	for _, role := range claimedRoles {
		if slices.Contains(allowedRoles, role) {
			return true
		}
	}
	return false
}
