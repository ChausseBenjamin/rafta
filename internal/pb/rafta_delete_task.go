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
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *raftaServer) DeleteTask(ctx context.Context, task *m.UUID) (*emptypb.Empty, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	taskID, err := uuid.Parse(task.Value)
	if err != nil {
		slog.ErrorContext(ctx, "failed to task uuid",
			logging.ErrKey, err,
		)
		return nil, status.Error(
			codes.Internal,
			"failure while parsing task id",
		)
	}

	rowCount, err := s.db.DeleteUserTask(ctx, database.DeleteUserTaskParams{
		Owner: creds.UserID,
		Task:  taskID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete task",
			logging.ErrKey, err,
		)
		return nil, status.Error(
			codes.Internal,
			"failed to delete task",
		)
	}
	if rowCount == 0 {
		slog.WarnContext(ctx, "no task got deleted")
		return nil, status.Errorf(
			codes.NotFound,
			"couldn't find task '%v' to delete it", taskID,
		)
	}

	go s.cleanTags(ctx)

	slog.InfoContext(ctx, "success")
	return &emptypb.Empty{}, nil
}
