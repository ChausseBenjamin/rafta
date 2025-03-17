package pb

import (
	"context"
	"log/slog"

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

	newTask := tx.Stmt(s.store.Common[db.CreateTask])
	_, err = newTask.ExecContext(ctx,
		taskID,
		task.Title,
		task.Priority,
		task.Desc,
		task.DueDate,
		task.DoDate,
		task.Recurrence.Pattern,
		task.Recurrence.Active,
		creds.UserID,
	)
	if err != nil {
		tx.Rollback()
		slog.ErrorContext(ctx, "Failed to insert task into the database", logging.ErrKey, err)
		return nil, status.Error(codes.Internal, "A failure occured while creating the task")
	}

	return &m.Task{
		Id:   &m.UUID{Value: taskID},
		Data: task,
	}, nil
}
