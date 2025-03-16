package pb

import (
	"context"
	"slices"

	"github.com/ChausseBenjamin/rafta/internal/db"
)

// getUserRoles is meant to be used when creating JWT tokens.
// Since login and signup only provide an email as an identifier, role
// information has to be fetched from the database.
func (s *AuthServer) getUserRoles(ctx context.Context, userID string) ([]string, error) {
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

func hasRequiredRole(claimedRoles []string, allowedRoles []string) bool {
	for _, role := range claimedRoles {
		if slices.Contains(allowedRoles, role) {
			return true
		}
	}
	return false
}
