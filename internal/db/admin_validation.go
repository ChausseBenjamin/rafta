package db

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/hashicorp/go-uuid"
)

const (
	defaultAdminName  = "Default Admin"
	defaultAdminEmail = "admin@localhost"
	defaultAdminRole  = "ADMIN"
)

// nolint:errcheck
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
