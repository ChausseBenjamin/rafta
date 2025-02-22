package pb

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/intercept"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc"
)

var ErrOutOfBoundsPort = errors.New("given port is out of bounds (1024-65535)")

// Implements ComsServer interface
type UserServer struct {
	db *db.Store
	m.UnimplementedRaftaUserServer
}

type AdminServer struct {
	db *db.Store
	m.UnimplementedRaftaAdminServer
}

func NewUserServer(store *db.Store) *UserServer {
	return &UserServer{db: store}
}

func NewAdminServer(store *db.Store) *AdminServer {
	return &AdminServer{db: store}
}

// Setup creates a new gRPC with both services
// and starts listening on the given port
func Setup(ctx context.Context, store *db.Store) (*grpc.Server, error) {
	slog.DebugContext(ctx, "Configuring gRPC server")
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		intercept.Tagging,
	))

	m.RegisterRaftaUserServer(server, NewUserServer(store))
	m.RegisterRaftaAdminServer(server, NewAdminServer(store))

	return server, nil
}
