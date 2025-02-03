package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/server"
	"github.com/ChausseBenjamin/rafta/internal/storage"
	"github.com/urfave/cli/v3"
)

func action(ctx context.Context, cmd *cli.Command) error {
	err := logging.Setup(
		cmd.String(FlagLogLevel),
		cmd.String(FlagLogFormat),
		cmd.String(FlagLogOutput),
	)
	if err != nil {
		slog.Warn("Error(s) occured during logger initialization", logging.ErrKey, err)
	}

	slog.Info("Starting rafta server")

	// TODO: Setup the db
	store, err := storage.Setup(cmd.String(FlagDBPath))
	if err != nil {
		slog.Error("Unable to setup database", logging.ErrKey, err)
	}

	srv, lis, err := server.Setup(cmd.Int(FlagListenPort), store)
	if err != nil {
		slog.Error("Unable to setup server", logging.ErrKey, err)

		return err
	}

	slog.Info(fmt.Sprintf("Listening on port %d", cmd.Int(FlagListenPort)))
	if err := srv.Serve(lis); err != nil {
		slog.Error("Server runtime error", logging.ErrKey, err)

		return err
	}

	return nil
}
