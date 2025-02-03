package util

type ContextKey uint8

const (
	DBKey ContextKey = iota
	ReqIDKey
)
