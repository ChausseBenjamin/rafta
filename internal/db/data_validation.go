package db

import (
	"context"
	"database/sql"
	"log/slog"
)

const (
	PublicUserSignupKey = "ACCEPT_PUBLIC_USERS"
	EnforceHttpsKey     = "ENFORCE_HTTPS"
	MaxUsersKey         = "MAX_USERS"
)

var settingValidations = [...]struct {
	key        string // key to look for in the settings
	defaultVal string // Default value to init if not set
}{
	{ // Only admins can create users when false
		PublicUserSignupKey,
		"FALSE",
	},
	{ // If something like traefik manages https, this can be set to
		// false. But there MUST be https in your stack otherwise
		// credentials are sent in the clear
		EnforceHttpsKey,
		"TRUE",
	},
	{ // Safeguard to avoid account creation spamming.
		// An admin can still create users over the limit
		MaxUsersKey,
		"25",
	},
}

func ValidateSettings(ctx context.Context, db *sql.DB) error {
	valTx, err := db.PrepareContext(ctx, "SELECT value FROM Settings WHERE key=?")
	if err != nil {
		return err
	}
	defer valTx.Close()

	newTx, err := db.PrepareContext(ctx, "INSERT INTO Settings (key, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer newTx.Close()

	for _, s := range settingValidations {
		var val string
		err := valTx.QueryRowContext(ctx, s.key).Scan(&val)
		if err != nil {
			return err
		}
		if val == "" {
			slog.WarnContext(ctx, "Missing configuration, setting the default",
				"setting", s.key,
				"value", s.defaultVal,
			)
			_, err := newTx.ExecContext(ctx, s.key, s.defaultVal)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
