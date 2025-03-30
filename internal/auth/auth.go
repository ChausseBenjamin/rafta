// auth handles everything related to JWT.
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

	"github.com/ChausseBenjamin/rafta/internal/database"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/sec"
	"github.com/ChausseBenjamin/rafta/internal/secrets"
	"github.com/ChausseBenjamin/rafta/internal/util"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// TODO: Make this the server url (configurable cli.Flag)
	issuer = "rafta-server"

	AccessTokenType  tokenType = "access"
	RefreshTokenType tokenType = "refresh"
	BasicTokenType   tokenType = "basic"
)

var (
	errTokenAlg     = errors.New("received token uses an unsupported signing method")
	errPrivKeyStore = errors.New("failed to store private key in vault")
	errPubKeyStore  = errors.New("failed to store public key in vault")
)

type AuthManager struct {
	pubKey  secrets.Secret
	privKey secrets.Secret
	db      *database.Queries
	cfg     *util.ConfigStore
}

type tokenType string

type Claims struct {
	Roles []string  `json:"roles"`
	Type  tokenType `json:"typ"`
	jwt.RegisteredClaims
}

// Credendials are what get sent to the protobuf server. This is done to
// minimize parsing by already having required fields in the uuid.UUID format
type Credendials struct {
	Subject uuid.UUID
	ID      uuid.UUID
	Claims
}

// Authenticating returns an interceptor that retrieves a jwt from the header.
// It must insert an object where the UserID and his roles are available down
// the line to other services.
func (a *AuthManager) Authenticating() grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
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
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid basic auth encoding: %v", err)
	}
	decodedStr := strings.TrimSpace(string(decodedCreds))

	credsParts := strings.SplitN(decodedStr, ":", 2)
	if len(credsParts) != 2 {
		return nil, status.Errorf(codes.Unauthenticated, "invalid basic auth format")
	}

	userSecret, err := a.db.GetUserSecretsFromEmail(ctx, credsParts[0])
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "User does not exist")
		}
		return nil, status.Error(codes.Internal, "Failed to query user credentials")
	}
	err = sec.ValidateCreds(credsParts[1], userSecret.Hash, userSecret.Salt)
	if err != nil {
		if errors.Is(err, sec.ErrInvalidCreds) {
			return nil, status.Error(codes.Unauthenticated, "Invalid credentials provided")
		}
		return nil, status.Error(codes.Unauthenticated, "Invalid credentials format provided")
	}

	// No errors mean successful hash validation, user credentials are stored in a "fake" jwt token
	// to streamline behaviour.
	roles, err := a.db.GetUserRoles(ctx, userSecret.UserID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.Internal, "Failed to retrieve user roles")
		}
		roles = []string{}
	}

	creds := &Credendials{
		Subject: userSecret.UserID,
		Claims: Claims{
			Roles: roles,
			Type:  BasicTokenType,
		},
	}

	newCtx := context.WithValue(ctx, util.CredsKey, creds)
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
		slog.WarnContext(ctx, "Unable to extract custom claims from JWT")
		return handler(ctx, req)
	} else {
		tokenID, err := util.ParseUUID(ctx, util.ParseUUIDParams{
			Str: tokenWithClaims.ID, Subject: "jwt_id",
			Implication: codes.Unauthenticated, Critical: true,
		})
		if err != nil {
			return nil, err
		}
		revoked, err := a.db.TokenIsRevoked(ctx, tokenID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to ensure provided token is not revoked",
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

		userID, err := util.ParseUUID(ctx, util.ParseUUIDParams{
			Str: tokenWithClaims.Subject, Subject: "jwt_id",
			Implication: codes.Unauthenticated, Critical: true,
		})
		if err != nil {
			return nil, err
		}

		creds := &Credendials{
			Subject: userID,
			ID:      tokenID,
			Claims:  *tokenWithClaims,
		}

		return handler(context.WithValue(ctx, util.CredsKey, creds), req)
	}
}

// Issue generates and returns a new access token and refresh token for the given user ID and roles.
// It returns the access token string, refresh token string, and an error if any occurs during the process.
func (a *AuthManager) Issue(userID uuid.UUID, roles []string) (string, string, error) {
	now := time.Now().UTC()
	accessID := uuid.New()
	refreshID := uuid.New()

	// Create the accessClaims
	accessClaims := Claims{
		Roles: roles,
		Type:  AccessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(a.cfg.JWTAccessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        accessID.String(),
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
		Roles: roles,
		Type:  RefreshTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(a.cfg.JWTRefreshTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        refreshID.String(),
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

// Since there are some encpoints that don't require authentication (Signup)
// The JWT interceptor can let unauthenticated request pass through. Not
// catching this can (and will) lead to nil pointer dereferences. This function
// returns a protobuf-ready error to allow auth-dependent endpoints to use the
// following pattern:
//
//	creds, err := auth.GetCreds(ctx, TokenBasicName)
//	if err != nil {
//		// err -> pre-formatted for protobuf with status code
//		return nil, err
//	}
func GetCreds(ctx context.Context, expects tokenType) (*Credendials, error) {
	creds := util.GetFromContext[Credendials](ctx, util.CredsKey)
	if creds == nil {
		slog.WarnContext(ctx,
			"User is not authenticated, cannot proceed with request",
		)
		return nil, status.Error(codes.Unauthenticated,
			"Current endpoint requires JWT authentication to proceed and found none. Operation aborted",
		)
	} else {
		if creds.Type != expects {
			slog.WarnContext(ctx,
				"User provided the wrong type of token, cannot proceed with request",
			)
			return nil, status.Errorf(codes.InvalidArgument,
				"Invalid token type for this endpoint. Expected: '%s', Got: '%s'", expects, creds.Type,
			)
		}
		return creds, nil
	}
}

func NewManager(vault secrets.SecretVault, db *database.Queries, cfg *util.ConfigStore) (*AuthManager, error) {
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
			return nil, errPubKeyStore
		}
		if err := vault.Set("server-privkey", secrets.Secret(privateKey)); err != nil {
			return nil, errPrivKeyStore
		}

		pubkey = secrets.Secret(publicKey)
		privkey = secrets.Secret(privateKey)
	}

	return &AuthManager{
		pubKey:  pubkey,
		privKey: privkey,
		db:      db,
		cfg:     cfg,
	}, nil
}

func (a *AuthManager) RevokeToken(ctx context.Context, token *Claims) error {
	tokenID, err := uuid.Parse(token.ID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse token ID", logging.ErrKey, err)
		return status.Error(codes.InvalidArgument, "Invalid token ID")
	}

	err = a.db.RevokeToken(ctx, database.RevokeTokenParams{
		TokenID: tokenID,
		Expiry:  token.ExpiresAt.Time.UTC(),
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to revoke token", logging.ErrKey, err)
		return status.Error(codes.Internal, "Token revocation failure")
	}

	err = a.db.CleanupExpiredToken(ctx, tokenID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to cleanup expired token", logging.ErrKey, err)
		// client shouldn't be concerned with internal housekeeping, don't return an error
	}

	return nil
}

func (a *AuthManager) ValidatePasswd(p string) error {
	// Password length
	if l := len(p); l < a.cfg.MinPasswdLen || l > a.cfg.MaxPasswdLen {
		return status.Errorf(codes.InvalidArgument,
			"Provided password is of length %d which is outside of the accepted range [%d-%d]",
			l, a.cfg.MinPasswdLen, a.cfg.MaxPasswdLen,
		)
	}
	// Illegal password characters
	for _, r := range p {
		if r < 32 || r > 126 {
			return status.Errorf(codes.InvalidArgument,
				"Provided password contains illegal characters. Allowed characters are in the [32-126] range (https://www.ascii-code.com)",
			)
		}
	}
	return nil
}
