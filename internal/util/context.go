package util

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
