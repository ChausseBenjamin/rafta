package util

import (
	"context"
	"log/slog"

	"github.com/golang-jwt/jwt/v5"
)

type ContextKey uint8

const (
	DBKey ContextKey = iota
	ReqIDKey
	ProtoMethodKey
	ProtoServerKey
	CredentialsKey
)

type ConfigStore struct {
	AllowNewUsers bool
	MaxUsers      int
	MinPasswdLen  int
	MaxPasswdLen  int
}

func GetToken(ctx context.Context) *jwt.Token {
	if value := ctx.Value(CredentialsKey); value != nil {
		if token, ok := value.(*jwt.Token); ok {
			return token
		}
	}
	slog.WarnContext(ctx, "Failed to retrieve JWT from context")
	return nil
}
