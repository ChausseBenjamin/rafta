package pb

import (
	"context"
	"log/slog"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *raftaServer) GetAllTasks(ctx context.Context, _ *emptypb.Empty) (*m.TaskList, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		return nil, err
	}
	var tasks []*m.Task
	stmt := s.store.Common[db.GetUserTasks]
	rows, err := stmt.QueryContext(ctx, creds.UserID)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failure to query tasks for a given user",
			logging.ErrKey, err,
		)
		return nil, status.Error(codes.Internal,
			"Failed to fetch user tasks internally",
		)
	}
	for rows.Next() {
		var (
			taskID            string
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
		if err := rows.Scan(
			&taskID,
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
			slog.ErrorContext(ctx,
				"Error during scan of user tasks",
				logging.ErrKey, err,
			)
			return nil, status.Error(codes.Internal,
				"Failure while scanning for user tasks",
			)
		}

		tasks = append(tasks, &m.Task{
			Id: &m.UUID{Value: taskID},
			Data: &m.TaskData{
				Title:    title,
				Priority: priority,
				Desc:     desc,
				DueDate:  timestamppb.New(dueDate),
				DoDate:   timestamppb.New(doDate),
				Recurrence: &m.TaskRecurrence{
					Pattern: recurrencePattern,
					Active:  recurrenceActive,
				},
			},
			Metadata: &m.TaskMetadata{
				CreatedOn: timestamppb.New(created),
				UpdatedOn: timestamppb.New(updated),
			},
		})
	}

	return &m.TaskList{
		Tasks: tasks,
	}, nil
}
