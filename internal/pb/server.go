// server.go Creates and instantiates all 3 protobuf servers that rafta uses:
//   - Admin
//   - Auth
//   - Rafta
package pb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/database"
	"github.com/ChausseBenjamin/rafta/internal/intercept"
	"github.com/ChausseBenjamin/rafta/internal/util"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var ErrOutOfBoundsPort = errors.New("given port is out of bounds (1024-65535)")

var adminRoles []string = []string{"ADMIN"}

// embedding here lets server use the `db` keyword to access both DB methods
// and Queries as if they were the same. It feels natural to be able to do both
// `s.db.BeginTX(...)` and `s.db.NewUser(...)` from the same place.
type protoDB struct {
	*sql.DB
	*database.Queries
}

// used to simplify wrapping for certain tasks
type protoServer struct {
	auth *auth.AuthManager
	cfg  *util.ConfigStore
	db   *protoDB
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
	*protoServer
	m.UnimplementedAuthServer
}

func NewRaftaServer(ps *protoServer) *raftaServer {
	return &raftaServer{protoServer: ps}
}

func NewAdminServer(ps *protoServer) *adminServer {
	return &adminServer{protoServer: ps}
}

func NewAuthServer(ps *protoServer) *authServer {
	return &authServer{protoServer: ps}
}

// Setup creates a new gRPC with both services
// and starts listening on the given port
func Setup(ctx context.Context, authMgr *auth.AuthManager, cfg *util.ConfigStore, db *sql.DB) (*grpc.Server, *database.Queries, error) {
	slog.DebugContext(ctx, "Configuring gRPC server")
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		intercept.Tagging,
		authMgr.Authenticating(),
	))

	queries, err := database.Prepare(ctx, db)
	if err != nil {
		return nil, nil, err
	}

	ps := &protoServer{auth: authMgr, cfg: cfg, db: &protoDB{DB: db, Queries: queries}}

	reflection.Register(server)
	m.RegisterAuthServer(server, NewAuthServer(ps))
	m.RegisterAdminServer(server, NewAdminServer(ps))
	m.RegisterRaftaServer(server, NewRaftaServer(ps))

	return server, queries, nil
}
