package db

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/hashicorp/go-uuid"
)

const (
	PublicUserSignupKey = "ACCEPT_PUBLIC_USERS"
	EnforceHttpsKey     = "ENFORCE_HTTPS"
	MaxUsersKey         = "MAX_USERS"

	defaultAdminName  = "Default Admin"
	defaultAdminEmail = "admin@localhost"
	defaultAdminRole  = "ADMIN"
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

func validateSettings(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	existsStmt, err := tx.PrepareContext(ctx, "SELECT EXISTS(SELECT 1 FROM Settings WHERE key=?)")
	if err != nil {
		return err
	}
	defer existsStmt.Close()

	newStmt, err := tx.PrepareContext(ctx, "INSERT INTO Settings (key, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer newStmt.Close()

	for _, s := range settingValidations {
		var exists bool
		err := existsStmt.QueryRowContext(ctx, s.key).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			slog.WarnContext(ctx, "Missing configuration, setting the default",
				"setting", s.key,
				"value", s.defaultVal,
			)
			_, err := newStmt.ExecContext(ctx, s.key, s.defaultVal)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//nolint:errcheck
func validateAdmin(ctx context.Context, db *sql.DB) error {
	// Begin transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Check if the ADMIN role exists
	var exists bool
	err = tx.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM Roles WHERE role = ?)",
		defaultAdminRole,
	).Scan(&exists)
	if err != nil {
		tx.Rollback()
		return err
	}

	// If the ADMIN role does not exist, create it
	if !exists {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO Roles (role) VALUES (?)",
			defaultAdminRole,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
		slog.Info("ADMIN role created")
	}

	// Check the count of users with the ADMIN role
	var adminCount int
	err = tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM UserRoles WHERE role = ?",
		defaultAdminRole,
	).Scan(&adminCount)
	if err != nil {
		tx.Rollback()
		return err
	}

	// If no admin users exist, create a default admin user
	if adminCount < 1 {
		// Generate a UUID for the admin
		userID, err := uuid.GenerateUUID()
		if err != nil {
			tx.Rollback()
			return err
		}

		// Generate a password for the admin
		passwd, saltedHash, err := auth.GenPassword()
		if err != nil {
			slog.ErrorContext(ctx,
				"Failed to generate a password for the new service amin user",
			)
			return err
		}
		slog.WarnContext(ctx, `No ADMIN found within the database.
		Creating one now with the following credentials.
		It is HIGHLY recommended you change the admin password or create AS SOON AS POSSIBLE.
		Also, clear these logs and restart your service to minimize credentials exposure.
		`,
			"admin_name", defaultAdminName,
			"admin_email", defaultAdminEmail,
			"admin_passwd", passwd,
		)

		// Insert into Users table
		_, err = tx.ExecContext(ctx,
			"INSERT INTO Users (userID, name, email) VALUES (?, ?, ?)",
			userID,
			defaultAdminName,
			defaultAdminEmail,
		)
		if err != nil {
			tx.Rollback()
			return err
		}

		// Insert into UserSecrets table
		_, err = tx.ExecContext(ctx,
			"INSERT INTO UserSecrets (userID, saltAndHash) VALUES (?, ?)",
			userID,
			saltedHash,
		)
		if err != nil {
			tx.Rollback()
			return err
		}

		// Insert into UserRoles table
		_, err = tx.ExecContext(ctx,
			"INSERT INTO UserRoles (userID, role) VALUES (?, ?)",
			userID,
			defaultAdminRole,
		)
		if err != nil {
			tx.Rollback()
			return err
		}

		slog.Info("Initialized new admin user")
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
