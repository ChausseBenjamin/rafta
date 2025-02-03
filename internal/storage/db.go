package storage

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/util"
)

func GetDB(ctx context.Context) *sql.DB {
	db, ok := ctx.Value(util.DBKey).(*sql.DB)
	if !ok {
		slog.Error("Unable to retrieve database from context")
		return nil
	}
	return db
}
