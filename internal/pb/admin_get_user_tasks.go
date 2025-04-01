package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/util"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
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

	userID, err := util.ParseUUID(ctx, util.ParseUUIDParams{
		Str: id.Value, Subject: "user_id",
		Critical: true, Implication: codes.InvalidArgument,
	})
	if err != nil {
		return nil, err
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

	slog.InfoContext(ctx, "success", "user_id", creds.Subject)
	return &m.TaskList{
		Tasks: tasksPb,
	}, nil
}
