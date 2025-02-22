package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/pb"
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
		slog.WarnContext(ctx, "Error(s) occurred during logger initialization", logging.ErrKey, err)
	}
	slog.InfoContext(ctx, "Starting rafta server")

	errAppChan := make(chan error)
	readyChan := make(chan bool)
	shutdownDone := make(chan struct{}) // Signals when graceful shutdown is complete

	var once sync.Once
	gracefulShutdown := func() {}
	brutalShutdown := func() {}

	application := func() {
		server, store, err := initApp(ctx, cmd)
		if err != nil {
			errAppChan <- err
			return
		}

		gracefulShutdown = func() {
			once.Do(func() { // Ensure brutal shutdown isn't triggered later
				server.GracefulStop()
				store.Close()
				slog.InfoContext(ctx, "Application shutdown")
				close(shutdownDone) // Signal that graceful shutdown is complete
			})
		}

		brutalShutdown = func() {
			once.Do(func() { // Ensure graceful shutdown isn't re-executed
				slog.WarnContext(ctx, "Graceful shutdown delay exceeded, shutting down NOW!")
				server.Stop()
				store.Close()
			})
		}

		port := fmt.Sprintf(":%d", cmd.Int(FlagListenPort))
		listener, err := net.Listen("tcp", port)
		if err != nil {
			errAppChan <- err
			return
		}
		slog.InfoContext(ctx, "Server listening", "port", cmd.Int(FlagListenPort))
		readyChan <- true

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
			case <-shutdownDone: // If graceful shutdown completes in time, exit normally
			case <-time.After(cmd.Duration(FlagGraceTimeout)): // Timeout exceeded
				brutalShutdown()
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

func initApp(ctx context.Context, cmd *cli.Command) (*grpc.Server, *db.Store, error) {
	store, err := db.Setup(ctx, cmd.String(FlagDBPath))
	if err != nil {
		slog.ErrorContext(ctx, "Unable to setup database", logging.ErrKey, err)
		return nil, nil, err
	}

	server, err := pb.Setup(ctx, store)
	if err != nil {
		slog.ErrorContext(ctx, "Unable to setup gRPC server", logging.ErrKey, err)
		return nil, nil, err
	}

	return server, store, nil
}
