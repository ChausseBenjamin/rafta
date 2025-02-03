package main

import (
	"log/slog"
	"os"

	"github.com/ChausseBenjamin/rafta/internal/app"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	docs "github.com/urfave/cli-docs/v3"
)

func main() {
	a := app.Command()

	man, err := docs.ToManWithSection(a, 1)
	if err != nil {
		slog.Error("failed to generate markdown", logging.ErrKey, err)
		os.Exit(1)
	}
	os.Stdout.Write([]byte(man))
}
