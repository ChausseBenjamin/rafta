package server

import (
	"context"
	"log/slog"

	m "github.com/ChausseBenjamin/rafta/internal/server/model"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s Service) GetUserTasks(ctx context.Context, id *m.UserID) (*m.TaskList, error) {
	slog.ErrorContext(ctx, "GetUserTasks not implemented yet")
	return nil, nil
}

func (s Service) GetTask(ctx context.Context, id *m.TaskID) (*m.Task, error) {
	return nil, nil
}

func (s Service) DeleteTask(ctx context.Context, id *m.TaskID) (*emptypb.Empty, error) {
	slog.ErrorContext(ctx, "DeleteTask not implemented yet")
	return nil, nil
}

func (s Service) UpdateTask(ctx context.Context, t *m.Task) (*m.Task, error) {
	slog.ErrorContext(ctx, "UpdateTask not implemented yet")
	return t, nil
}

func (s Service) CreateTask(ctx context.Context, data *m.TaskData) (*m.Task, error) {
	slog.ErrorContext(ctx, "CreateTask not implemented yet")
	return nil, nil
}
