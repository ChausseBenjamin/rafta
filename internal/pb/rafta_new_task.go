package pb

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	"github.com/ChausseBenjamin/rafta/internal/database"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *raftaServer) NewTask(ctx context.Context, t *m.TaskData) (*m.NewTaskResponse, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to start task creation transaction",
			logging.ErrKey, err,
		)
		return nil, status.Error(
			codes.Internal,
			"Failed to begin user creation",
		)
	}
	defer tx.Rollback()

	db := s.db.WithTx(tx)

	task, err := db.NewTask(ctx, database.NewTaskParams{
		Title:    t.Title,
		State:    uint8(t.State),
		Priority: t.Priority,
		Description: sql.NullString{
			String: t.Desc,
			Valid:  (t.Desc != ""),
		},
		DueDate: t.DueDate.AsTime().UTC(),
		DoDate:  t.DoDate.AsTime().UTC(),
		RecurrencePattern: sql.NullString{
			String: t.Recurrence.Pattern,
			Valid:  (t.Recurrence.Pattern != ""),
		},
		RecurrenceEnabled: t.Recurrence.Active,
		Owner:             creds.UserID,
	})
	if err != nil {
		slog.ErrorContext(ctx,
			"failed to insert task into database",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"Failed to insert task",
		)
	}

	if len(t.Tags) > 0 {
		err = s.syncTags(ctx, task.TaskID, t.Tags, db)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx,
			"failed to commit transaction",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"failed to properly complete task creation",
		)
	}
	tx = nil

	// cleanup shouldn't block the transaction as it's just housekeeping. That's
	// why it's done outside of syncTags and after the transaction completes
	go s.cleanTags(ctx)

	slog.InfoContext(ctx, "success")
	return &m.NewTaskResponse{
		Id: &m.UUID{Value: task.TaskID.String()},
		Metadata: &m.TaskMetadata{
			CreatedOn: timestamppb.New(task.CreatedAt.UTC()),
			UpdatedOn: timestamppb.New(task.UpdatedAt.UTC()),
		},
	}, nil
}
