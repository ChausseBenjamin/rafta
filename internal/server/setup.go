package server

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net"

	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/internal/server/model"
	"github.com/ChausseBenjamin/rafta/internal/tagging"
	"google.golang.org/grpc"
)

func Setup(port int64, storage *sql.DB) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen(
		"tcp",
		fmt.Sprintf(":%d", port),
	)
	if err != nil {
		slog.Error("Unable to create listener", logging.ErrKey, err)
		return nil, nil, err
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			tagging.UnaryInterceptor,
		),
	)
	raftaService := &Service{store: storage}
	m.RegisterRaftaServer(grpcServer, raftaService)

	return grpcServer, lis, nil
}
