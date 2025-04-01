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

func (s *raftaServer) UpdateTask(ctx context.Context, req *m.TaskUpdateRequest) (*m.TaskUpdateResponse, error) {
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

	q := bqb.New("update tasks set updated_at = CURRENT_TIMESTAMP")
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
		title,
		state,
		priority,
		description,
		due_date,
		do_date,
		recurrence_pattern,
		recurrence_enabled,
		updated_on;`, req.Id, creds.ID,
	).ToSql()
	if err != nil {
		slog.ErrorContext(ctx, "failed to build query",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "failed to build task update")
	}

	var (
		recurrenceEnabled bool
		recurrencePattern string
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
		)
		return nil, status.Error(codes.Internal,
			"failed to feetch updated task",
		)
	}

	if recurrenceEnabled {
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

func (s *raftaServer) rescheduleTask(ctx context.Context, taskID uuid.UUID, tx *sql.Tx) error {
	return status.Error(codes.Unimplemented, "Recurrence isn't configured yet, aborting")
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
