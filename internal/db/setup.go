package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrForeignKeysDisabled  = errors.New("foreign keys are disabled")
	ErrIntegrityCheckFailed = errors.New("integrity check failed")
	ErrJournalModeInvalid   = errors.New("journal mode is not WAL")
	ErrSchemaMismatch       = errors.New("database schema does not match expected definition")
	ErrTableMissing         = errors.New("table is missing")
	ErrTableStructure       = errors.New("table structure does not match expected schema")
)

// Store contains both a database and a list all prepared statements the
// server needs to use at runtime.
// The benefits of centralizing all statements are twofold:
//
//  1. It's easy to have an overview of how data gets queried
//  2. Since statements are sanitized to avoid SQL injection, this security is
//     inherent to all calls made to the database
//
// The only reason DB is actually needed here is for the auth package (which
// would cause a circular dependency) and the use of transactions where it
// can be useful to dismiss multiple statements at once if something
// unexpected happens.
type Store struct {
	DB     *sql.DB
	Common []*sql.Stmt
}

func new(db *sql.DB) (*Store, error) {
	lst := make([]*sql.Stmt, len(commonStatements))
	for _, common := range commonStatements {
		stmt, err := db.Prepare(common.Cmd)
		if err != nil {
			return nil, err
		}
		lst[common.Name] = stmt
	}
	return &Store{
		DB:     db,
		Common: lst,
	}, nil
}

func (s *Store) Close() error {
	errs := make([]error, len(s.Common)+1)
	for i, s := range s.Common {
		if s != nil {
			errs[i] = s.Close()
		}
	}
	errs[len(s.Common)] = s.DB.Close()
	return errors.Join(errs...)
}

// opts returns connection options that enforce our desired pragmas.
func opts() string {
	return "?_foreign_keys=on&_journal_mode=WAL"
}

// Setup opens the SQLite DB at path, verifies its integrity and schema,
// and returns the valid DB handle. If any check fails, it backs up the old
// file and reinitializes the DB using the schema definitions.
func Setup(ctx context.Context, path string) (*Store, error) {
	slog.DebugContext(ctx, "Setting up database connection")
	var db *sql.DB
	var err error
	var integrity string

	// If file does not exist, generate a new DB.
	if _, statErr := os.Stat(path); statErr != nil {
		var genErr error
		db, genErr = genDB(ctx, path)
		if genErr != nil {
			return nil, genErr
		}
	} else {
		db, err = sql.Open("sqlite", path+opts())
		if err != nil {
			slog.ErrorContext(ctx, "failed to open DB", "error", err)
			backupFile(ctx, path)
			db, err = genDB(ctx, path)
		}
	}

	if err == nil {
		_, err = db.Exec("PRAGMA foreign_keys = ON")
	}

	if err == nil {
		_, err = db.Exec("PRAGMA journal_mode=WAL")
	}

	if err == nil {
		queryErr := db.QueryRow("PRAGMA integrity_check;").Scan(&integrity)
		if queryErr != nil || integrity != "ok" {
			if queryErr != nil {
				slog.ErrorContext(ctx, "integrity check query failed", "error", queryErr)
			} else {
				slog.ErrorContext(ctx, "integrity check failed", "integrity", integrity)
			}
			db.Close()
			backupFile(ctx, path)
			db, err = genDB(ctx, path)
		}
	}

	if err == nil {
		schemaErr := validateSchema(ctx, db)
		if schemaErr != nil {
			slog.ErrorContext(ctx, "schema validation failed", "error", schemaErr)
			db.Close()
			backupFile(ctx, path)
			db, err = genDB(ctx, path)
		}
	}

	if err == nil {
		adminErr := validateAdmin(ctx, db)
		if adminErr != nil {
			err = adminErr
		}
	}

	if err != nil {
		return nil, err
	}

	return new(db)
}

// validateSchema checks that the PRAGMAs and every table definition match the expected schema.
func validateSchema(ctx context.Context, db *sql.DB) error {
	if err := validatePragmas(db); err != nil {
		return err
	}
	for _, table := range schemaDefinitions {
		if err := validateTable(ctx, db, table.Name, table.Cmd); err != nil {
			return err
		}
	}
	return nil
}

// validatePragmas ensures that the required PRAGMAs are set.
func validatePragmas(db *sql.DB) error {
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&fk); err != nil {
		return err
	}
	if fk != 1 {
		return ErrForeignKeysDisabled
	}

	var jm string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&jm); err != nil {
		return err
	}
	if strings.ToLower(jm) != "wal" {
		return ErrJournalModeInvalid
	}
	return nil
}

// validateTable fetches the stored SQL for the table and compares it
// (after normalization) with the expected definition.
func validateTable(ctx context.Context, db *sql.DB, tableName, expectedSQL string) error {
	actualSQL, err := fetchTableSQL(db, tableName)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch table definition", "table", tableName, "error", err)
		return ErrSchemaMismatch
	}
	if actualSQL == "" {
		slog.ErrorContext(ctx, "table is missing", "table", tableName)
		return ErrTableMissing
	}

	normalizedExpected := normalizeSQL(expectedSQL)
	normalizedActual := normalizeSQL(actualSQL)
	if normalizedExpected != normalizedActual {
		slog.ErrorContext(ctx, "table structure does not match expected schema",
			"table", tableName,
			"expected", normalizedExpected,
			"actual", normalizedActual,
		)
		return ErrTableStructure
	}
	return nil
}

// normalizeSQL removes SQL comments, converts to lowercase,
// collapses whitespace, and removes a trailing semicolon.
func normalizeSQL(sqlStr string) string {
	sqlStr = removeSQLComments(sqlStr)
	sqlStr = strings.ToLower(sqlStr)
	sqlStr = strings.ReplaceAll(sqlStr, "\n", " ")
	sqlStr = strings.Join(strings.Fields(sqlStr), " ")
	sqlStr = strings.TrimSuffix(sqlStr, ";")
	return sqlStr
}

// removeSQLComments strips out any '--' style comments.
func removeSQLComments(sqlStr string) string {
	lines := strings.Split(sqlStr, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "--"); idx != -1 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, " ")
}

// fetchTableSQL retrieves the SQL definition of a table from sqlite_master.
func fetchTableSQL(db *sql.DB, table string) (string, error) {
	var sqlDef sql.NullString
	err := db.QueryRow(
		"SELECT sql FROM sqlite_master WHERE type='table' AND name=?",
		table,
	).Scan(&sqlDef)
	if err != nil {
		return "", err
	}
	if !sqlDef.Valid {
		return "", fmt.Errorf("no SQL definition found for table %s", table)
	}
	return sqlDef.String, nil
}

// backupFile renames the existing file by appending a ".bak" (or timestamped) suffix.
func backupFile(ctx context.Context, path string) {
	backupPath := path + ".bak"
	if _, err := os.Stat(backupPath); err == nil {
		backupPath = fmt.Sprintf("%s-%s.bak", path, time.Now().UTC().Format(time.RFC3339))
	}
	if err := os.Rename(path, backupPath); err != nil {
		slog.ErrorContext(ctx, "failed to backup file",
			"error", err,
			"original", path,
			"backup", backupPath,
		)
	} else {
		slog.InfoContext(ctx, "backed up corrupt DB",
			"original", path,
			"backup", backupPath,
		)
	}
}
