package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/pb"
	"github.com/urfave/cli/v3"
)

// Avoids string mismatches when calling cmd.String(), cmd.Int(), etc...
const (
	FlagConfigPath       = "config"
	FlagDBPath           = "database"
	FlagDisableHTTPS     = "disable-https"
	FlagDisablePubSignup = "disable-public-signups"
	FlagGraceTimeout     = "grace-timeout"
	FlagListenPort       = "port"
	FlagLogFormat        = "log-format"
	FlagLogLevel         = "log-level"
	FlagLogOutput        = "log-output"
	FlagMaxUsers         = "max-users"
	FlagSecretsPath      = "secrets-path"
)

func flags() []cli.Flag {
	return []cli.Flag{
		// Logging {{{
		&cli.StringFlag{
			Name:    FlagLogFormat,
			Aliases: []string{"f"},
			Value:   "plain",
			Usage:   "plain, json",
			Sources: cli.EnvVars("LOG_FORMAT"),
			Action:  validateLogFormat,
		},
		&cli.StringFlag{
			Name:    FlagLogOutput,
			Aliases: []string{"o"},
			Value:   "stdout",
			Usage:   "stdout, stderr, file",
			Sources: cli.EnvVars("LOG_OUTPUT"),
			Action:  validateLogOutput,
		},
		&cli.StringFlag{
			Name:    FlagLogLevel,
			Aliases: []string{"l"},
			Value:   "info",
			Usage:   "debug, info, warn, error",
			Sources: cli.EnvVars("LOG_LEVEL"),
			Action:  validateLogLevel,
		}, // }}}
		// gRPC {{{
		&cli.IntFlag{
			Name:    FlagListenPort,
			Aliases: []string{"p"},
			Value:   1157, // list in leetspeak :P
			Sources: cli.EnvVars("LISTEN_PORT"),
			Action:  validateListenPort,
		},
		&cli.BoolFlag{ // TODO: Implement https
			Name:    FlagDisableHTTPS,
			Value:   false,
			Usage:   `Disable secure https communication. WARNING: Be very careful using this. Only do this if your server is behind a reverse proxy that already handles https for it and you trust all network communications on that network.`,
			Sources: cli.EnvVars("DISABLE_HTTPS"),
		},
		&cli.DurationFlag{
			Name:    FlagGraceTimeout,
			Aliases: []string{"t"},
			Value:   5 * time.Second,
			Sources: cli.EnvVars("GRACEFUL_TIMEOUT"),
		}, // }}}
		// Database {{{
		&cli.StringFlag{
			Name:    FlagDBPath,
			Aliases: []string{"d"},
			Value:   "store.db",
			Usage:   "database file",
			Sources: cli.EnvVars("DATABASE_PATH"),
		}, // }}}
		// Service {{{
		&cli.StringFlag{
			Name:    FlagSecretsPath,
			Value:   "/etc/secrets",
			Usage:   "Directory containing necessary secrets (ca_certs, private keys, etc...)",
			Sources: cli.EnvVars("SECRETS_PATH"),
		},
		&cli.UintFlag{ // TODO: uttilize MAX_USERS
			Name:    FlagMaxUsers,
			Value:   25,
			Usage:   "Maximum number of users that can get created without admin intervention",
			Sources: cli.EnvVars("MAX_USERS"),
		},
		&cli.BoolFlag{ // TODO: uttilize DISABLE_PUBLIC_SIGNUP
			Name:    FlagDisablePubSignup,
			Usage:   "Deactivate public (non admin-based) signups",
			Sources: cli.EnvVars("DISABLE_PUBLIC_SIGNUP"),
		}, // }}}
	}
}

func validateLogOutput(ctx context.Context, cmd *cli.Command, s string) error {
	switch {
	case s == "stdout" || s == "stderr":
		return nil
	default:
		// assume file
		f, err := os.OpenFile(s, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			slog.ErrorContext(
				ctx,
				fmt.Sprintf("Error creating/accessing provided log file %s", s),
			)
			return err
		}
		defer f.Close()
		return nil
	}
}

func validateLogLevel(ctx context.Context, cmd *cli.Command, s string) error {
	for _, lvl := range []string{"deb", "inf", "warn", "err"} {
		if strings.Contains(strings.ToLower(s), lvl) {
			return nil
		}
	}
	slog.ErrorContext(
		ctx,
		fmt.Sprintf("Unknown log level provided: %s", s),
	)
	return logging.ErrInvalidLevel
}

func validateLogFormat(ctx context.Context, cmd *cli.Command, s string) error {
	s = strings.ToLower(s)
	if s == "json" || s == "plain" {
		return nil
	}
	return nil
}

func validateListenPort(ctx context.Context, cmd *cli.Command, p int64) error {
	if p < 1024 || p > 65535 {
		slog.ErrorContext(
			ctx,
			fmt.Sprintf("Out-of-bound port provided: %d", p),
		)
		return pb.ErrOutOfBoundsPort
	}
	return nil
}
