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
)

var ErrOutOfBoundsPort = errors.New("given port is out of bounds (1024-65535)")

type RaftaServer struct {
	store *db.Store
	cfg   *util.ConfigStore
	m.UnimplementedRaftaServer
}

type AdminServer struct {
	store *db.Store
	cfg   *util.ConfigStore
	m.UnimplementedAdminServer
}

type AuthServer struct {
	store   *db.Store
	authMgr *auth.AuthManager // To issue tokens
	cfg     *util.ConfigStore
	m.UnimplementedAuthServer
}

func NewRaftaServer(store *db.Store, cfg *util.ConfigStore) *RaftaServer {
	return &RaftaServer{store: store, cfg: cfg}
}

func NewAdminServer(store *db.Store, cfg *util.ConfigStore) *AdminServer {
	return &AdminServer{store: store, cfg: cfg}
}

func NewAuthServer(store *db.Store, authMgr *auth.AuthManager, cfg *util.ConfigStore) *AuthServer {
	return &AuthServer{store: store, authMgr: authMgr, cfg: cfg}
}

// Setup creates a new gRPC with both services
// and starts listening on the given port
func Setup(ctx context.Context, store *db.Store, vault secrets.SecretVault, cfg *util.ConfigStore) (*grpc.Server, error) {
	authMgr, err := auth.NewManager(vault, store.DB)
	if err != nil {
		return nil, err
	}

	slog.DebugContext(ctx, "Configuring gRPC server")
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		intercept.Tagging,
		authMgr.Authenticating(),
	))

	m.RegisterAuthServer(server, NewAuthServer(store, authMgr, cfg))
	m.RegisterAdminServer(server, NewAdminServer(store, cfg))
	m.RegisterRaftaServer(server, NewRaftaServer(store, cfg))

	return server, nil
}
