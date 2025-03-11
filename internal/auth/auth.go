package auth

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/logging"
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
	accessTokenDuration  = 20 * time.Hour // for ease of testing
	accessTokenName      = "access"
	refreshTokenName     = "refresh"
)

var (
	errTokenAlg     = errors.New("Received token uses an unsupported signing method")
	privKeyStoreErr = errors.New("failed to store private key in vault")
	pubKeyStoreErr  = errors.New("failed to store public key in vault")
	uuidErr         = errors.New("Failed to generate a UUID for a token")
)

type AuthManager struct {
	pubKey      secrets.Secret
	privKey     secrets.Secret
	db          *sql.DB
	revokeCheck *sql.Stmt
}

type Claims struct {
	UserID string   `json:"uuid"`
	Roles  []string `json:"roles"`
	Type   string   `json:"token_type"`
	jwt.RegisteredClaims
}

// Authenticating returns an interceptor that retrieves a jwt from the header.
// It must insert an object where the UserID and his roles are available down
// the line to other services.
func (a *AuthManager) Authenticating() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		tokenMetadata := md["authorization"]
		if len(tokenMetadata) == 0 {
			return handler(ctx, req)
		}

		authHeader := tokenMetadata[0]
		switch strings.ToLower(strings.Split(tokenMetadata[0], " ")[0]) {
		case "bearer":
			return a.handleBearerAuth(ctx, req, handler, authHeader)
		case "basic":
			return a.handleBasicAuth(ctx, req, handler, authHeader)
		default:
			return nil, status.Errorf(codes.Unauthenticated, "unsupported authorization method")
		}
	}
}

func (a *AuthManager) handleBasicAuth(ctx context.Context, req any, handler grpc.UnaryHandler, authHeader string) (any, error) {
	encodedCreds := strings.TrimPrefix(authHeader, "Basic ")
	decodedCreds, err := base64.StdEncoding.DecodeString(encodedCreds)
	decodedStr := strings.TrimSpace(string(decodedCreds))
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid basic auth encoding: %v", err)
	}

	credsParts := strings.SplitN(decodedStr, ":", 2)
	if len(credsParts) != 2 {
		return nil, status.Errorf(codes.Unauthenticated, "invalid basic auth format")
	}

	creds := &Credentials{
		Email:  credsParts[0],
		Secret: secrets.Secret(credsParts[1]),
	}

	newCtx := context.WithValue(ctx, util.CredentialsKey, creds)
	return handler(newCtx, req)
}

func (a *AuthManager) handleBearerAuth(ctx context.Context, req any, handler grpc.UnaryHandler, authHeader string) (any, error) {
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			slog.InfoContext(ctx, "Received token uses an unsupported signing method", "algorithm", token.Header["alg"])
			return nil, errTokenAlg
		}
		return ed25519.PublicKey(a.pubKey.Bytes()), nil
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	if tokenWithClaims, ok := token.Claims.(*Claims); !ok {
		slog.Warn("Unable to extract custom claims from JWT")
		return handler(ctx, req)
	} else {
		revoked := false
		row := a.revokeCheck.QueryRowContext(ctx, tokenWithClaims.ID)
		err := row.Scan(&revoked)
		if err != nil {
			slog.Error("failed to ensure provided token is not revoked",
				logging.ErrKey, err,
			)
			return nil, status.Error(
				codes.Internal,
				"failure while ensuring token isn't revoked",
			)
		}
		if revoked {
			return nil, status.Error(codes.Unauthenticated,
				"provided token has been revoked",
			)
		}

		return handler(context.WithValue(ctx, util.JwtKey, tokenWithClaims), req)
	}
}

func (a *AuthManager) Issue(userID string, roles []string) (string, string, error) {
	now := time.Now().UTC()
	accessID, accessIDErr := uuid.GenerateUUID()
	refreshID, refreshIDErr := uuid.GenerateUUID()
	if accessIDErr != nil || refreshIDErr != nil {
		return "", "", uuidErr
	}

	// Create the accessClaims
	accessClaims := Claims{
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
	refreshClaims := Claims{
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

	// The validation check is not set in the db package to avoid circular
	// dependencies. This should be the only one though
	stmt, err := db.Prepare(
		"SELECT EXISTS(SELECT 1 FROM RevokedTokens WHERE tokenID = ?)",
	)
	if err != nil {
		slog.Error("Failed to prepare revocation check query for JWT",
			logging.ErrKey, err,
		)
	}

	return &AuthManager{
		db:          db,
		pubKey:      pubkey,
		privKey:     privkey,
		revokeCheck: stmt,
	}, nil
}

func (a *AuthManager) Close() {
	err := a.revokeCheck.Close()
	if err != nil {
		slog.Error("Failure closing the token revocation check query for JWT",
			logging.ErrKey, err,
		)
	}
}
