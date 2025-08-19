package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/database"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/pb"
	"github.com/ChausseBenjamin/rafta/internal/secrets"
	"github.com/ChausseBenjamin/rafta/internal/util"
	"github.com/urfave/cli/v3"
	"google.golang.org/grpc"
)

func action(ctx context.Context, cmd *cli.Command) error {
	err := logging.Setup(
		cmd.String(FlagLogLevel),
		cmd.String(FlagLogFormat),
		cmd.String(FlagLogOutput),
	)
	if err != nil {
		slog.WarnContext(ctx, "Error(s) occurred during logger initialization",
			logging.ErrKey, err,
		)
	}
	slog.InfoContext(ctx, "Starting rafta server")

	errAppChan := make(chan error)
	shutdownDone := make(chan struct{}) // Signals when graceful shutdown is done

	var once sync.Once
	gracefulShutdown := func() {}
	brutalShutdown := func() {}

	application := func() {
		server, db, queries, err := initApp(ctx, cmd)
		if err != nil {
			errAppChan <- err
			return
		}

		//nolint:errcheck
		gracefulShutdown = func() {
			once.Do(func() { // Ensure brutal shutdown isn't triggered later
				server.GracefulStop()
				db.Close()
				queries.Close()
				slog.InfoContext(ctx, "Application shutdown")
				close(shutdownDone) // Signal that graceful shutdown is complete
			})
		}

		//nolint:errcheck
		brutalShutdown = func() {
			slog.WarnContext(ctx,
				"Graceful shutdown delay exceeded, shutting down NOW!",
			)
			server.Stop()
			db.Close()
			queries.Close()
		}

		port := fmt.Sprintf(":%d", cmd.Int(FlagListenPort))
		listener, err := net.Listen("tcp", port)
		if err != nil {
			errAppChan <- err
			return
		}
		slog.InfoContext(ctx, "Server listening", "port", cmd.Int(FlagListenPort))

		if err := server.Serve(listener); err != nil {
			errAppChan <- err
		}
	}
	go application()

	stopChan := waitForTermChan()
	running := true
	for running {
		select {
		case errApp := <-errAppChan:
			if errApp != nil {
				slog.ErrorContext(ctx, "Application error", logging.ErrKey, errApp)
			}
			return errApp
		case <-stopChan:
			slog.InfoContext(ctx, "Shutdown requested")
			go gracefulShutdown()

			select {
			case <-time.After(cmd.Duration(FlagGraceTimeout)): // Timeout exceeded
				brutalShutdown()
			case <-shutdownDone: // If graceful shutdown is timely, exit normally
			}
			running = false
		}
	}
	return nil
}

func waitForTermChan() chan os.Signal {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	return stopChan
}

// func initApp(ctx context.Context, cmd *cli.Command) (*grpc.Server, *db.Store, *auth.AuthManager, error) {
func initApp(ctx context.Context, cmd *cli.Command) (*grpc.Server, *sql.DB, *database.Queries, error) {
	globalConf := &util.ConfigStore{
		AllowNewUsers: !cmd.Bool(FlagDisablePubSignup),
		MaxUsers:      int(cmd.Uint(FlagMaxUsers)),
		MinPasswdLen:  int(cmd.Uint(FlagMinPasswdLen)),
		MaxPasswdLen:  int(cmd.Uint(FlagMaxPasswdLen)),
		JWTAccessTTL:  cmd.Duration(FlagAccessTokenTTL),
		JWTRefreshTTL: cmd.Duration(FlagRefreshTokenTTL),
		DBCacheSize:   int(-cmd.Uint(FlagDBCacheSize)),
		ArgonThreads:  uint(cmd.Uint(FlagArgonThreads)),
	}

	vault, err := secrets.NewDirVault(cmd.String(FlagSecretsPath))
	if err != nil {
		return nil, nil, nil, err
	}

	db, err := database.Setup(ctx, cmd.String(FlagDBPath), globalConf)
	if err != nil {
		return nil, nil, nil, err
	}

	authMgr, err := auth.NewManager(vault, database.New(db), globalConf)
	if err != nil {
		return nil, nil, nil, err
	}

	server, queries, err := pb.Setup(ctx, authMgr, globalConf, db)
	if err != nil {
		slog.ErrorContext(ctx, "Unable to setup gRPC server", logging.ErrKey, err)
		return nil, nil, nil, err
	}

	return server, db, queries, nil
}
