package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *adminServer) GetUserTasks(ctx context.Context, id *m.UUID) (*m.TaskList, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	if err := s.hasAdminRights(ctx, creds); err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(id.Value)
	if err != nil {
		slog.WarnContext(ctx,
			"failed to parse provided userID",
			"user_id", id.Value,
			logging.ErrKey, err,
		)
		return nil, status.Errorf(codes.InvalidArgument,
			"Failed to parse provided user id. Parser returned '%v'", err,
		)
	}

	tasks, err := s.db.GetUserTasks(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failure to retrieve task for user",
			"user_id", userID,
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"Couldn't retrieve user tasks",
		)
	}

	tasksPb := make([]*m.Task, len(tasks))

	for i, task := range tasks {
		tags, err := s.db.GetTaskTags(ctx, task.TaskID)
		if err != nil {
			slog.ErrorContext(ctx,
				"Failed to query tags for a specific task",
				"task_id", task.TaskID,
				logging.ErrKey, err,
			)
			return nil, status.Errorf(codes.Internal,
				"Failed to query tags for task '%v'", task.TaskID,
			)
		}

		tasksPb[i] = taskToPb(task, tags)
	}

	slog.InfoContext(ctx, "success", "user_id", creds.UserID)
	return &m.TaskList{
		Tasks: tasksPb,
	}, nil
}
