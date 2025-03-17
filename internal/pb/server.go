package pb

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/intercept"
	"github.com/ChausseBenjamin/rafta/internal/secrets"
	"github.com/ChausseBenjamin/rafta/internal/util"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var ErrOutOfBoundsPort = errors.New("given port is out of bounds (1024-65535)")

type raftaServer struct {
	store *db.Store
	cfg   *util.ConfigStore
	m.UnimplementedRaftaServer
}

type adminServer struct {
	store *db.Store
	cfg   *util.ConfigStore
	m.UnimplementedAdminServer
}

type authServer struct {
	store   *db.Store
	authMgr *auth.AuthManager // To issue tokens
	cfg     *util.ConfigStore
	m.UnimplementedAuthServer
}

func NewRaftaServer(store *db.Store, cfg *util.ConfigStore) *raftaServer {
	return &raftaServer{store: store, cfg: cfg}
}

func NewAdminServer(store *db.Store, cfg *util.ConfigStore) *adminServer {
	return &adminServer{store: store, cfg: cfg}
}

func NewAuthServer(store *db.Store, authMgr *auth.AuthManager, cfg *util.ConfigStore) *authServer {
	return &authServer{store: store, authMgr: authMgr, cfg: cfg}
}

// Setup creates a new gRPC with both services
// and starts listening on the given port
func Setup(ctx context.Context, store *db.Store, vault secrets.SecretVault, cfg *util.ConfigStore) (*grpc.Server, *auth.AuthManager, error) {
	authMgr, err := auth.NewManager(vault, store.DB)
	if err != nil {
		return nil, nil, err
	}

	slog.DebugContext(ctx, "Configuring gRPC server")
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		intercept.Tagging,
		authMgr.Authenticating(),
	))

	reflection.Register(server)
	m.RegisterAuthServer(server, NewAuthServer(store, authMgr, cfg))
	m.RegisterAdminServer(server, NewAdminServer(store, cfg))
	m.RegisterRaftaServer(server, NewRaftaServer(store, cfg))

	return server, authMgr, nil
}
