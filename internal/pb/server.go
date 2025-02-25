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
	store *db.Store
	m.UnimplementedRaftaServer
}

type AdminServer struct {
	store *db.Store
	m.UnimplementedAdminServer
}

type AuthServer struct {
	store *db.Store
	m.UnimplementedAuthServer
}

func NewRaftaServer(store *db.Store) *UserServer {
	return &UserServer{store: store}
}

func NewAdminServer(store *db.Store) *AdminServer {
	return &AdminServer{store: store}
}

func NewAuthServer(store *db.Store) *AuthServer {
	return &AuthServer{store: store}
}

// Setup creates a new gRPC with both services
// and starts listening on the given port
func Setup(ctx context.Context, store *db.Store) (*grpc.Server, error) {
	slog.DebugContext(ctx, "Configuring gRPC server")
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		intercept.Tagging,
	))

	m.RegisterAuthServer(server, NewAuthServer(store))
	m.RegisterAdminServer(server, NewAdminServer(store))
	m.RegisterRaftaServer(server, NewRaftaServer(store))

	return server, nil
}
