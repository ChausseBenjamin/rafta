package server

import (
	"database/sql"
	"errors"

	m "github.com/ChausseBenjamin/rafta/internal/server/model"
)

var ErrOutOfBoundsPort = errors.New("port out of bounds")

// Implements ComsServer interface
type Service struct {
	store *sql.DB
	m.UnimplementedRaftaServer
}
