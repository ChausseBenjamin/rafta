package pb

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/intercept"
	"github.com/ChausseBenjamin/rafta/internal/secrets"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc"
)

var ErrOutOfBoundsPort = errors.New("given port is out of bounds (1024-65535)")

type UserServer struct {
	store *db.Store
	m.UnimplementedRaftaServer
}

type AdminServer struct {
	store *db.Store
	m.UnimplementedAdminServer
}

type AuthServer struct {
	store   *db.Store
	authMgr *auth.AuthManager // To issue tokens
	m.UnimplementedAuthServer
}

func NewRaftaServer(store *db.Store) *UserServer {
	return &UserServer{store: store}
}

func NewAdminServer(store *db.Store) *AdminServer {
	return &AdminServer{store: store}
}

func NewAuthServer(store *db.Store, authMgr *auth.AuthManager) *AuthServer {
	return &AuthServer{store: store, authMgr: authMgr}
}

// Setup creates a new gRPC with both services
// and starts listening on the given port
func Setup(ctx context.Context, store *db.Store, vault secrets.SecretVault) (*grpc.Server, error) {
	authMgr, err := auth.NewManager(vault, store.DB)
	if err != nil {
		return nil, err
	}

	slog.DebugContext(ctx, "Configuring gRPC server")
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		intercept.Tagging,
		authMgr.Authenticating(),
	))

	m.RegisterAuthServer(server, NewAuthServer(store, authMgr))
	m.RegisterAdminServer(server, NewAdminServer(store))
	m.RegisterRaftaServer(server, NewRaftaServer(store))

	return server, nil
}
