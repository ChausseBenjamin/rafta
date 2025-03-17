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

// used to simplify wrapping for certain tasks
type protoServer struct {
	store *db.Store
	cfg   *util.ConfigStore
}

type raftaServer struct {
	*protoServer
	m.UnimplementedRaftaServer
}

type adminServer struct {
	*protoServer
	m.UnimplementedAdminServer
}

type authServer struct {
	authMgr *auth.AuthManager
	*protoServer
	m.UnimplementedAuthServer
}

func NewRaftaServer(ps *protoServer) *raftaServer {
	return &raftaServer{protoServer: ps}
}

func NewAdminServer(ps *protoServer) *adminServer {
	return &adminServer{protoServer: ps}
}

func NewAuthServer(ps *protoServer, authMgr *auth.AuthManager) *authServer {
	return &authServer{protoServer: ps, authMgr: authMgr}
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

	ps := &protoServer{cfg: cfg, store: store}

	reflection.Register(server)
	m.RegisterAuthServer(server, NewAuthServer(ps, authMgr))
	m.RegisterAdminServer(server, NewAdminServer(ps))
	m.RegisterRaftaServer(server, NewRaftaServer(ps))

	return server, authMgr, nil
}
