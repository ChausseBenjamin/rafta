package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrIntegrityCheckFailed = errors.New("integrity check failed")
	ErrForeignKeysDisabled  = errors.New("foreign_keys pragma is not enabled")
	ErrJournalModeInvalid   = errors.New("journal_mode is not wal")
	ErrSchemaMismatch       = errors.New("schema does not match expected definition")
)

// Setup opens the SQLite DB at path, verifies its integrity and schema,
// and returns the valid DB handle. On any error, it backs up the old file
// (if it exists) and calls genDB() to initialize a valid schema.
func Setup(path string) (*sql.DB, error) {
	_, statErr := os.Stat(path)
	exists := statErr == nil

	// If file doesn't exist, generate new DB.
	if !exists {
		return genDB(path)
	}

	db, err := sql.Open("sqlite3", path+opts())
	if err != nil {
		slog.Error("failed to open DB", "error", err)
		backupFile(path)
		return genDB(path)
	}

	// Integrity check.
	var integrity string
	if err = db.QueryRow("PRAGMA integrity_check;").Scan(&integrity); err != nil {
		slog.Error("integrity check query failed", "error", err)
		db.Close()
		backupFile(path)
		return genDB(path)
	}
	if integrity != "ok" {
		slog.Error("integrity check failed", "error", ErrIntegrityCheckFailed)
		db.Close()
		backupFile(path)
		return genDB(path)
	}

	// Validate schema and pragmas.
	if err = validateSchema(db); err != nil {
		slog.Error("schema validation failed", "error", err)
		db.Close()
		backupFile(path)
		return genDB(path)
	}

	return db, nil
}

// validateSchema verifies that required pragmas and table definitions are set.
func validateSchema(db *sql.DB) error {
	// Check PRAGMA foreign_keys = on.
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&fk); err != nil {
		return err
	}
	if fk != 1 {
		return ErrForeignKeysDisabled
	}

	// Check PRAGMA journal_mode = wal.
	var jm string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&jm); err != nil {
		return err
	}
	if strings.ToLower(jm) != "wal" {
		return ErrJournalModeInvalid
	}

	// Define required table definitions (as substrings in lower-case).
	type tableCheck struct {
		name       string
		substrings []string
	}

	checks := []tableCheck{
		{
			name: "User",
			substrings: []string{
				"create table user",
				"userid", "integer", "primary key", "autoincrement",
				"name", "text", "not null",
				"email", "text", "not null", "unique",
			},
		},
		{
			name: "Task",
			substrings: []string{
				"create table task",
				"taskid", "integer", "primary key", "autoincrement",
				"title", "not null",
				"description", "not null",
				"due", "date", "not null",
				"do", "date", "not null",
				"owner", "integer", "not null",
				"foreign key", "references user",
			},
		},
		{
			name: "Tag",
			substrings: []string{
				"create table tag",
				"tagid", "integer", "primary key", "autoincrement",
				"name", "text", "not null", "unique",
			},
		},
		{
			name: "TaskTag",
			substrings: []string{
				"create table tasktag",
				"taskuuid", "integer", "not null",
				"tagid", "integer", "not null",
				"primary key",
				"foreign key", "references task",
				"foreign key", "references tag",
			},
		},
	}

	for _, chk := range checks {
		sqlDef, err := fetchTableSQL(db, chk.name)
		if err != nil {
			return fmt.Errorf("failed to fetch definition for table %s: %w", chk.name, err)
		}
		lc := strings.ToLower(sqlDef)
		for _, substr := range chk.substrings {
			if !strings.Contains(lc, substr) {
				return fmt.Errorf("%w: table %s missing %q", ErrSchemaMismatch, chk.name, substr)
			}
		}
	}

	return nil
}

func fetchTableSQL(db *sql.DB, table string) (string, error) {
	var sqlDef sql.NullString
	err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&sqlDef)
	if err != nil {
		return "", err
	}
	if !sqlDef.Valid {
		return "", fmt.Errorf("no SQL definition found for table %s", table)
	}
	return sqlDef.String, nil
}

// backupFile renames the existing file by appending a ".bak" suffix.
func backupFile(path string) {
	backupPath := path + ".bak"
	// If backupPath exists, append a timestamp.
	if _, err := os.Stat(backupPath); err == nil {
		backupPath = fmt.Sprintf("%s.%d.bak", path, os.Getpid())
	}
	if err := os.Rename(path, backupPath); err != nil {
		slog.Error("failed to backup file", "error", err, "original", path, "backup", backupPath)
	} else {
		slog.Info("backed up corrupt DB", "original", path, "backup", backupPath)
	}
}

// genDB creates a new database at path with the valid schema.
func genDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		slog.Error("failed to create DB", "error", err)
		return nil, err
	}

	// Set pragmas.
	if _, err := db.Exec("PRAGMA foreign_keys = on; PRAGMA journal_mode = wal;"); err != nil {
		slog.Error("failed to set pragmas", "error", err)
		db.Close()
		return nil, err
	}

	if _, err := db.Exec(schema()); err != nil {
		slog.Error("failed to initialize schema", "error", err)
		db.Close()
		return nil, err
	}

	slog.Info("created new blank DB with valid schema", "path", path)
	return db, nil
}
