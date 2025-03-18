package pb

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *raftaServer) CreateTask(ctx context.Context, task *m.TaskData) (*m.Task, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		return nil, err
	}

	taskID, err := s.generateUniqueUUID(ctx, s.store.Common[db.AssertTaskExists])
	if err != nil {
		return nil, err
	}

	tx, err := s.store.DB.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to start task insertion transaction",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "Failed to start task insertion")
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	_, err = tx.Stmt(s.store.Common[db.CreateTask]).ExecContext(ctx,
		taskID,
		task.Title,
		task.Priority,
		task.Desc,
		task.DueDate.AsTime(),
		task.DoDate.AsTime(),
		task.Recurrence.Pattern,
		task.Recurrence.Active,
		now, // created
		now, // updated
		creds.UserID,
	)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to insert task into the database",
			"user", creds.UserID,
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"A failure occured while creating the task",
		)
	}

	if len(task.Tags) > 0 {
		err := s.syncTags(ctx, tx, taskID, task.Tags)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		slog.ErrorContext(ctx,
			"There was an error commiting the new task",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "Failed to finish task insertion")
	}
	return &m.Task{
		Id:   &m.UUID{Value: taskID},
		Data: task,
	}, nil
}

func (s *protoServer) syncTags(ctx context.Context, tx *sql.Tx, taskID string, tags []string) error {
	// Get current tags for the task.
	existing := make(map[string]int64)
	rows, err := tx.Stmt(s.store.Common[db.GetTaskTags]).QueryContext(
		ctx,
		taskID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var tagID int64
		if err = rows.Scan(&name, &tagID); err != nil {
			return err
		}
		existing[name] = tagID
	}
	if err = rows.Err(); err != nil {
		return err
	}

	// Build set of tags from the task struct.
	newSet := make(map[string]struct{})
	for _, tag := range tags {
		newSet[tag] = struct{}{}
	}

	// 2 & 3. Unlink tags not in newSet and delete unused ones.
	for name, tagID := range existing {
		if _, keep := newSet[name]; !keep {
			if _, err = tx.ExecContext(ctx,
				taskID, tagID); err != nil {
				return err
			}
			if _, err = tx.Stmt(s.store.Common[db.DeleteTaskTag]).ExecContext(ctx, tagID, tagID); err != nil {
				return err
			}
		}
	}

	// Link new tags: create missing tags and add to TaskTags.
	for _, tag := range tags {
		if _, exists := existing[tag]; !exists {
			// Create tag if not exists.
			if _, err = tx.Stmt(s.store.Common[db.CreateTag]).ExecContext(ctx, tag); err != nil {
				return err
			}
			var tagID int64
			if err = tx.Stmt(s.store.Common[db.GetTagID]).QueryRowContext(ctx, tag).Scan(&tagID); err != nil {
				return err
			}
			if _, err = tx.Stmt(s.store.Common[db.CreateTaskTag]).ExecContext(ctx, taskID, tagID); err != nil {
				return err
			}
		}
	}
	return nil
}
