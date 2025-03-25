package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/database"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *raftaServer) GetTask(ctx context.Context, id *m.UUID) (*m.Task, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	taskID, err := uuid.Parse(id.Value)
	if err != nil {
		slog.WarnContext(ctx,
			"failed to parse provided taskID",
			"task_id", id.Value,
			logging.ErrKey, err,
		)
		return nil, status.Errorf(codes.InvalidArgument,
			"Failed to parse provided task id. Parser returned '%v'", err,
		)
	}

	task, err := s.db.GetUserTask(ctx, database.GetUserTaskParams{
		TaskID: taskID,
		Owner:  creds.UserID,
	})
	if err != nil {
		slog.ErrorContext(ctx,
			"failed to retrieve task",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "failed to retrieve task")
	}

	tags, err := s.db.GetTaskTags(ctx, task.TaskID)
	if err != nil {
		slog.ErrorContext(ctx,
			"failed to retrieve tags associated with task",
			"task_id", task.TaskID,
			logging.ErrKey, err,
		)
		return nil, status.Errorf(codes.Internal,
			"Failure while retrieving tags associated with '%v'", task.TaskID,
		)
	}

	slog.InfoContext(ctx, "success")
	return taskToPb(task, tags), nil
}
