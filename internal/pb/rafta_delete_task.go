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

	if err := s.db.DeleteUserTask(ctx, database.DeleteUserTaskParams{
		Owner: creds.UserID,
		Task:  taskID,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to delete task",
			logging.ErrKey, err,
		)
		// TODO: differentiate between NotFound and Internal error type
		return nil, status.Error(
			codes.Internal,
			"failed to delete task",
		)
	}

	go s.cleanTags(ctx)

	slog.InfoContext(ctx, "success")
	return &emptypb.Empty{}, nil
}
