package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *raftaServer) DeleteTask(ctx context.Context, taskID *m.UUID) (*emptypb.Empty, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := s.store.DB.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to start task deletion transaction",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "Failed to start task deletion")
	}
	defer tx.Rollback()

	// Ensures unused tags get cleaned up
	err = s.syncTags(ctx, tx, taskID.Value, []string{})
	if err != nil {
		return nil, err
	}

	res, err := tx.Stmt(s.store.Common[db.DeleteUserTask]).ExecContext(ctx, taskID.Value, creds.UserID)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failure during task deletion",
			"task", taskID.Value,
			logging.ErrKey, err,
		)
		return nil, status.Errorf(codes.Internal,
			"Failed to delete task '%s'", taskID.Value,
		)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		slog.ErrorContext(ctx,
			"Failure checking if deletion request had an impact",
			"task", taskID.Value,
			logging.ErrKey, err,
		)
		return nil, status.Errorf(codes.Internal,
			"Failed to assert task deletion for '%s'", taskID.Value,
		)
	}
	if rowsAffected == 0 {
		slog.WarnContext(ctx,
			"Attempt made to delete a nonexistent task",
			"task", taskID.Value,
		)
		return nil, status.Errorf(codes.NotFound,
			"task '%s' not found", taskID.Value,
		)
	}

	return &emptypb.Empty{}, nil
}
