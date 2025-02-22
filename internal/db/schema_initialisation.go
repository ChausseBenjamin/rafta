package db

import (
	"context"
	"database/sql"
	"log/slog"
)

// schemaDefinitions is the single source of truth for both creating and validating the DB schema.
var schemaDefinitions = [...]struct {
	Name string
	Cmd  string
}{
	{
		"Users",
		`CREATE TABLE Users (
			userID TEXT PRIMARY KEY CHECK (length(userID) = 36),
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
	},
	{
		"UserSecrets",
		`CREATE TABLE UserSecrets (
			userID TEXT PRIMARY KEY,
			saltAndHash TEXT NOT NULL,
			FOREIGN KEY (userID) REFERENCES Users(userID) ON DELETE CASCADE
		);`,
	},
	{
		"Tasks",
		`CREATE TABLE Tasks (
			taskID TEXT PRIMARY KEY CHECK (length(taskID) = 36),
			title TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			description TEXT,
			due TIMESTAMP,
			do TIMESTAMP,
			cron TEXT,
			cronIsEnabled BOOLEAN NOT NULL DEFAULT FALSE,
			owner TEXT NOT NULL,
			FOREIGN KEY (owner) REFERENCES Users(userID) ON DELETE CASCADE
		);`,
	},
	{
		"Tags",
		`CREATE TABLE Tags (
			tagID INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);`,
	},
	{
		"TaskTags",
		`CREATE TABLE TaskTags (
			taskID TEXT NOT NULL,
			tagID INTEGER NOT NULL,
			PRIMARY KEY (taskID, tagID),
			FOREIGN KEY (taskID) REFERENCES Tasks(taskID) ON DELETE CASCADE,
			FOREIGN KEY (tagID) REFERENCES Tags(tagID) ON DELETE CASCADE
		);`,
	},
	{
		"Settings",
		`CREATE TABLE Settings (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
	},
	{
		"Roles",
		`CREATE TABLE Roles (
			role TEXT PRIMARY KEY CHECK (role GLOB '[A-Z_]*')
		);`,
	},
	{
		"UserRoles",
		`CREATE TABLE UserRoles (
			userID TEXT NOT NULL,
			role TEXT NOT NULL,
			PRIMARY KEY (userID, role),
			FOREIGN KEY (userID) REFERENCES Users(userID) ON DELETE CASCADE,
			FOREIGN KEY (role) REFERENCES Roles(role) ON DELETE CASCADE
		);`,
	},
}

// genDB creates a new database at path using the expected schema definitions.
func genDB(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create DB", "error", err)
		return nil, err
	}

	// Set the required PRAGMAs.
	if _, err := db.Exec("PRAGMA foreign_keys = on; PRAGMA journal_mode = wal;"); err != nil {
		slog.ErrorContext(ctx, "failed to set pragmas", "error", err)
		db.Close()
		return nil, err
	}

	// Create tables inside a transaction.
	tx, err := db.Begin()
	if err != nil {
		slog.ErrorContext(ctx, "failed to begin transaction for schema initialization", "error", err)
		db.Close()
		return nil, err
	}
	for _, table := range schemaDefinitions {
		if _, err := tx.Exec(table.Cmd); err != nil {
			slog.ErrorContext(ctx, "failed to initialize schema", "table", table.Name, "error", err)
			if errRollback := tx.Rollback(); errRollback != nil {
				slog.ErrorContext(ctx, "failed to rollback schema initialization", "error", errRollback)
			}

			db.Close()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx, "failed to commit schema initialization", "error", err)
		db.Close()
		return nil, err
	}

	slog.InfoContext(ctx, "created new blank DB wit h valid schema", "path", path)
	return db, nil
}
