package auth

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/secrets"
	"github.com/ChausseBenjamin/rafta/internal/util"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/go-uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// TODO: Make these configurable through flags/env variables
	issuer               = "rafta-server"
	refreshTokenDuration = 24 * time.Hour
	accessTokenDuration  = 20 * time.Minute
	accessTokenName      = "access"
	refreshTokenName     = "refresh"
)

var (
	privKeyStoreErr = errors.New("failed to store private key in vault")
	pubKeyStoreErr  = errors.New("failed to store public key in vault")
	uuidErr         = errors.New("Failed to generate a UUID for a token")
)

type AuthManager struct {
	pubKey  secrets.Secret
	privKey secrets.Secret
	db      *sql.DB
}

type claims struct {
	UserID string   `json:"uuid"`
	Roles  []string `json:"roles"`
	Type   string   `json:"token_type"`
	jwt.RegisteredClaims
}

type credentials struct {
	UserID string
	Roles  []string
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
	slog.Error("TOKEN VALIDATION IS NOT IMPLEMENTED YET...")
	return credentials{}, nil
}

func (a *AuthManager) Issue(userID string, roles []string) (string, string, error) {
	now := time.Now()
	accessID, accessIDErr := uuid.GenerateUUID()
	refreshID, refreshIDErr := uuid.GenerateUUID()
	if accessIDErr != nil || refreshIDErr != nil {
		return "", "", uuidErr
	}

	// Create the accessClaims
	accessClaims := claims{
		UserID: userID,
		Roles:  roles,
		Type:   accessTokenName,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			Audience:  []string{"your-app-audience"},
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        accessID,
		},
	}

	// Create the access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, accessClaims)
	accessTokenString, err := accessToken.SignedString(ed25519.PrivateKey(a.privKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %v", err)
	}

	// Create the refresh token claims
	refreshClaims := claims{
		UserID: userID,
		Roles:  roles,
		Type:   refreshTokenName,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        refreshID,
		},
	}

	// Create the refresh token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(ed25519.PrivateKey(a.privKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %v", err)
	}

	return accessTokenString, refreshTokenString, nil
}

// Renew
func (a *AuthManager) Renew(token jwt.Token) {
	slog.Error("TOKEN VALIDATION IS NOT IMPLEMENTED YET...")
}

// Revoke adds a tokens uuid to the database and start a goroutine to clean it
// up once it expires.
func (a *AuthManager) Revoke(uuid string) {
	panic("unimplemented")
}

func NewManager(vault secrets.SecretVault, db *sql.DB) (*AuthManager, error) {
	pubkey, pubErr := vault.Get("server-pubkey")
	privkey, privErr := vault.Get("server-privkey")
	if pubErr != nil || privErr != nil {
		// Generate a new ed25519 key pair
		publicKey, privateKey, keyErr := ed25519.GenerateKey(nil)
		if keyErr != nil {
			return nil, fmt.Errorf("failed to generate ed25519 key: %v", keyErr)
		}

		// Store the keys in the vault
		if err := vault.Set("server-pubkey", secrets.Secret(publicKey)); err != nil {
			return nil, pubKeyStoreErr
		}
		if err := vault.Set("server-privkey", secrets.Secret(privateKey)); err != nil {
			return nil, privKeyStoreErr
		}

		pubkey = secrets.Secret(publicKey)
		privkey = secrets.Secret(privateKey)
	}
	return &AuthManager{
		db:      db,
		pubKey:  pubkey,
		privKey: privkey,
	}, nil
}
