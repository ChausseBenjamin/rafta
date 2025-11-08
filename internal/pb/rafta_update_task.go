package pb

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"github.com/google/uuid"
	"github.com/nullism/bqb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *raftaServer) UpdateTask(
	ctx context.Context,
	req *m.TaskUpdateRequest,
) (*m.TaskUpdateResponse, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	taskID, err := uuid.Parse(req.Id.Value)
	if err != nil {
		slog.ErrorContext(ctx, "failed to task id",
			logging.ErrKey, err,
		)
		return nil, status.Error(
			codes.Internal,
			"failure while parsing task id",
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx,
			"Task update transaction initialization failure",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"Failed begin start the process of updating a task",
		)
	}
	defer tx.Rollback()

	var state_changed bool
	q := bqb.New("update tasks set updated_on = CURRENT_TIMESTAMP")
	masks := removeDuplicate(req.Masks)
	for _, mask := range masks {
		switch mask {
		case m.TaskFieldMask_TITLE:
			q.Concat(", title = ?", req.Data.Title)
		case m.TaskFieldMask_DESC:
			q.Concat(", description = ?", req.Data.Desc)
		case m.TaskFieldMask_PRIORITY:
			q.Concat(", priority = ?", req.Data.Desc)
		case m.TaskFieldMask_STATE:
			q.Concat(", state = ?", req.Data.Desc)
			state_changed = true
		case m.TaskFieldMask_RECURRENCE:
			q.Concat(", recurrence_pattern = ?, recurrence_enabled = ?",
				req.Data.Recurrence.Pattern, req.Data.Recurrence.Active,
			)
		case m.TaskFieldMask_TAGS:
			if err := s.syncTags(ctx, taskID, req.Data.Tags, s.db.WithTx(tx)); err != nil {
				return nil, err
			}
		}
	}

	query, args, err := q.Concat(` where task_id = ? and owner = ? returning
		recurrence_pattern,
		recurrence_enabled,
		updated_on;`, req.Id.Value, creds.Subject,
	).ToSql()
	if err != nil {
		slog.ErrorContext(ctx, "failed to build query",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "failed to build task update")
	}

	slog.InfoContext(ctx, "executing update query",
		"query", query,
		"task_id", req.Id.Value,
		"owner_id", creds.Subject,
	)

	// Debug: Check if task exists and who owns it
	var existingOwner string
	checkQuery := "SELECT owner FROM tasks WHERE task_id = ?"
	if err := tx.QueryRowContext(ctx, checkQuery, req.Id.Value).Scan(&existingOwner); err != nil {
		if err == sql.ErrNoRows {
			slog.WarnContext(ctx, "task not found", "task_id", req.Id.Value)
			return nil, status.Error(codes.NotFound, "task not found")
		} else {
			slog.ErrorContext(ctx, "error checking task existence", logging.ErrKey, err)
			return nil, status.Error(codes.Internal, "failed to check task existence")
		}
	} else {
		slog.InfoContext(ctx, "task exists",
			"task_id", req.Id.Value,
			"actual_owner", existingOwner,
			"requesting_user", creds.Subject,
		)
		if existingOwner != creds.Subject.String() {
			slog.WarnContext(ctx, "unauthorized task update attempt",
				"task_id", req.Id.Value,
				"actual_owner", existingOwner,
				"requesting_user", creds.Subject,
			)
			return nil, status.Error(
				codes.PermissionDenied,
				"you do not have permission to update this task",
			)
		}
	}

	var (
		recurrenceEnabled bool
		recurrencePattern sql.NullString
		updatedOn         time.Time
	)
	row := tx.QueryRowContext(ctx, query, args...)
	if err := row.Scan(
		&recurrencePattern,
		&recurrenceEnabled,
		&updatedOn,
	); err != nil {
		slog.ErrorContext(ctx,
			"failed to ingest task update",
			logging.ErrKey, err,
			"query", query,
			"task_id", req.Id.Value,
			"owner_id", creds.Subject,
		)
		return nil, status.Error(codes.Internal,
			"failed to feetch updated task",
		)
	}

	if recurrenceEnabled && state_changed {
		if err := s.rescheduleTask(ctx, taskID, tx); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx,
			"failure to commit task update transaction",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "failed to complete task update")
	}
	return nil, nil
}

func (s *raftaServer) rescheduleTask(ctx context.Context, id uuid.UUID, _ *sql.Tx) error {
	// With task scheduling being unimplemented at the moment, return an error
	// would block any task containing recurrence info of being updated/used.
	// nil is returned instead of UNIMPLEMENTED to allow users to still use those
	// tasks despite that.
	slog.WarnContext(ctx, "Recurrence isn't currently implemented, skipping reschedule",
		"task_id", id,
	)
	return nil
}

func removeDuplicate[T comparable](sliceList []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
