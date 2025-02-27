package auth

import (
	"context"
	"database/sql"

	"github.com/ChausseBenjamin/rafta/internal/secrets"
	"github.com/ChausseBenjamin/rafta/internal/util"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthManager struct {
	vault secrets.SecretVault
	db    *sql.DB
}

type credentials struct {
	claims jwt.RegisteredClaims
	roles  []string
}

// Authenticating returns an interceptor that retrieves a jwt from the header
// then passes it to the Validate function. If Validate returns no error,
// the credentials are added to the context with the util.CredentialsKey
func (a *AuthManager) Authenticating() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}

		token := md["authorization"]
		if len(token) == 0 {
			return handler(ctx, req)
		}

		creds, err := a.Validate(token[0])
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		newCtx := context.WithValue(ctx, util.CredentialsKey, creds)
		return handler(newCtx, req)
	}
}

func (a *AuthManager) Validate(token string) (credentials, error) {
	// TODO: Implement validation
	return credentials{}, nil
}

func (a *AuthManager) Issue() {
	panic("unimplemented")
}

func (a *AuthManager) Renew() {
	panic("unimplemented")
}

// Revoke adds a tokens uuid to the database and start a goroutine to clean it up
// once it expires.
func (a *AuthManager) Revoke(uuid string) {
	panic("unimplemented")
}

func NewManager(vault secrets.SecretVault, db *sql.DB) (*AuthManager, error) {
	return &AuthManager{
		vault: vault,
		db:    db,
	}, nil
}
