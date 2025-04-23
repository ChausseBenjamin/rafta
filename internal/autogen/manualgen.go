/*
 * This package isn't the actual rafta server.
 * To avoid importing packages which aren't needed at runtime,
 * some auto-generation functionnalities is offloaded to here so
 * it can be done with access to the rest of the code-base but
 * without bloating the final binary. For example,
 * generating bash+zsh auto-completion scripts isn't needed in
 * the final binary if those script are generated before hand.
 * Same gose for manpages. This file is meant to be run automatically
 * to easily package new releases.
 */
package main

import (
	"log/slog"
	"os"

	"github.com/ChausseBenjamin/rafta/internal/app"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	docs "github.com/urfave/cli-docs/v3"
)

//nolint:errcheck
func main() {
	a := app.Command()

	man, err := docs.ToManWithSection(a, 1)
	if err != nil {
		slog.Error("failed to generate man page",
			slog.Any(logging.ErrKey, err),
		)
		os.Exit(1)
	}
	os.Stdout.Write([]byte(man))
}
