package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *raftaServer) GetAllTasks(ctx context.Context, _ *emptypb.Empty) (*m.TaskList, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	tasks, err := s.db.GetUserTasks(ctx, creds.UserID)
	if err != nil {
		slog.ErrorContext(ctx,
			"failed to retrieve tasks for given user",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal, "failed to retrieve tasks")
	}

	tasksPb := make([]*m.Task, len(tasks))
	for i, task := range tasks {
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

		tasksPb[i] = taskToPb(task, tags)
	}

	slog.InfoContext(ctx, "success")
	return &m.TaskList{
		Tasks: tasksPb,
	}, nil
}
