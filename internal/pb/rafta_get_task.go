package pb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *raftaServer) GetTask(ctx context.Context, id *m.UUID) (*m.Task, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		return nil, err
	}

	row := s.store.Common[db.GetUserTask].QueryRowContext(
		ctx,
		creds.UserID,
		id.Value,
	)

	var (
		title             string
		priority          uint32
		desc              string
		recurrencePattern string
		recurrenceActive  bool
		dueDate           time.Time
		doDate            time.Time
		created           time.Time
		updated           time.Time
	)

	if err := row.Scan(
		&title,
		&priority,
		&desc,
		&dueDate,
		&doDate,
		&recurrencePattern,
		&recurrenceActive,
		&created,
		&updated,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(ctx, "User fetched a non-existent task")
			return nil, status.Error(codes.NotFound,
				"Requested task does not exist",
			)
		}
		slog.ErrorContext(ctx,
			"Failure while fetching specific task for user",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"Failure to fetch specific task",
		)
	}

	return &m.Task{
		Id: id,
		Data: &m.TaskData{
			Title:    title,
			Priority: priority,
			Desc:     desc,
			Recurrence: &m.TaskRecurrence{
				Pattern: recurrencePattern,
				Active:  recurrenceActive,
			},
			DueDate: timestamppb.New(dueDate),
			DoDate:  timestamppb.New(doDate),
		},
		Metadata: &m.TaskMetadata{
			CreatedOn: timestamppb.New(created),
			UpdatedOn: timestamppb.New(updated),
		},
	}, nil
}
