package pb

import (
	"context"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/golang-jwt/jwt/v5"
)

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

// validateEmail ensure a given string is a valid email
func (s *AuthServer) validateEmail(email string) bool {
func revokeTokens(tokens []jwt.Token) {
	panic("unimplemented")
}
