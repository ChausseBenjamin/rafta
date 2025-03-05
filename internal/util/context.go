package util

import (
	"context"
	"log/slog"
)

type ContextKey uint8

const (
	DBKey ContextKey = iota
	ReqIDKey
	ProtoMethodKey
	ProtoServerKey
	JwtKey         // Generated from Bearer tokens
	CredentialsKey // Generated from Basic tokens
)

type ConfigStore struct {
	AllowNewUsers bool
	MaxUsers      int
	MinPasswdLen  int
	MaxPasswdLen  int
}

func GetFromContext[T any](ctx context.Context, key any) *T {
	if value := ctx.Value(key); value != nil {
		if asserted, ok := value.(*T); ok {
			return asserted
		}
	}
	slog.WarnContext(ctx, "Failed to retrieve item from context", "key", key)
	return nil
}
