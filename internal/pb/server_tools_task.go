package pb

import (
	"context"
	"log/slog"
	"slices"

	"github.com/ChausseBenjamin/rafta/internal/database"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func taskToPb(t database.Task, tags []database.Tag) *m.Task {
	tagsStr := make([]string, len(tags))
	for i, tag := range tags {
		tagsStr[i] = tag.Name
	}
	return &m.Task{
		Id: &m.UUID{Value: t.TaskID.String()},
		Data: &m.TaskData{
			Title:    t.Title,
			Desc:     t.Description.String,
			Priority: t.Priority,
			Tags:     tagsStr,
			DoDate:   timestamppb.New(t.DoDate.UTC()),
			DueDate:  timestamppb.New(t.DueDate.UTC()),
			State:    m.TaskState(t.State),
			Recurrence: &m.TaskRecurrence{
				Pattern: t.RecurrencePattern.String,
				Active:  t.RecurrenceEnabled,
			},
		},
		Metadata: &m.TaskMetadata{
			CreatedOn: timestamppb.New(t.CreatedOn.UTC()),
			UpdatedOn: timestamppb.New(t.UpdatedOn.UTC()),
		},
	}
}

// syncTags ensures that tags for a task are up-to-date by taking in a task,
// it's old and new tags and perform the following:
// - Unassign from task tags that are no longer used
// - Create tags that don't get exist
// - Assign any tag not currently assigned to the task
// - Unassign tags that are no longer associated with the task
// - Delete tags that are no longer linked to any task
func (s *protoServer) syncTags(ctx context.Context, taskID uuid.UUID, tagNames []string, db *database.Queries) error {
	// Build a []database.Tag containing tags that already exists
	existingTags, err := db.GetExistingTags(ctx, tagNames)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to retrieve existing tags for task",
			"task_id", taskID,
			logging.ErrKey, err,
		)
		return status.Errorf(codes.Internal,
			"error fetching existing tags for task '%v'", taskID,
		)
	}

	// For every task that doesn't exist but is needed, create the tag
	for _, tag := range tagNames {
		if !slices.ContainsFunc(existingTags, func(t database.Tag) bool {
			return (t.Name == tag)
		}) {
			newTag, err := db.NewTag(ctx, tag)
			if err != nil {
				slog.ErrorContext(ctx,
					"Failed to create tag needed for task",
					"task_id", taskID,
					logging.ErrKey, err,
				)
				return status.Errorf(codes.Internal,
					"failed to create tag needed for task '%v'", taskID,
				)
			}
			existingTags = append(existingTags, newTag)
		}
	}

	// Get a list of every tag currently linked to the target task
	linkedTags, err := db.GetTaskTags(ctx, taskID)
	if err != nil {
		slog.ErrorContext(ctx,
			"Failed to create tags currently linked to task",
			"task_id", taskID,
			logging.ErrKey, err,
		)
		return status.Errorf(codes.Internal,
			"failed to retrieve tags currently assigned to task '%v'", taskID,
		)
	}

	// Link every tag in existingTags that isn't already linked to the task
	for _, tag := range existingTags {
		if !slices.ContainsFunc(linkedTags, func(t database.Tag) bool {
			return (t.TagID == tag.TagID)
		}) {
			if err := db.AssignTag(ctx, database.AssignTagParams{
				Task: taskID,
				Tag:  tag.TagID,
			}); err != nil {
				slog.ErrorContext(ctx,
					"Failed to assign tag to task",
					"task_id", taskID,
					"tag_id", tag.TagID,
					logging.ErrKey, err,
				)
				return status.Errorf(codes.Internal,
					"failed to assign tags '%v' to task '%v'", tag.TagID, taskID,
				)
			}
			linkedTags = append(linkedTags, tag)
		}
	}

	// Unassign every tag from linkedTags that isn't in existingTags
	var tagsToUnassign []int64
	for _, tag := range linkedTags {
		if !slices.ContainsFunc(existingTags, func(t database.Tag) bool {
			return (tag.TagID == t.TagID)
		}) {
			tagsToUnassign = append(tagsToUnassign, tag.TagID)
		}
	}
	if len(tagsToUnassign) > 0 {
		if err := db.UnassignTags(ctx, tagsToUnassign); err != nil {
			slog.ErrorContext(ctx,
				"Failed to unassign tag(s) from task",
				"task_id", taskID,
				logging.ErrKey, err,
			)
			return status.Errorf(codes.Internal,
				"failed to unassign tag(s) from task '%v'", taskID,
			)
		}
	}

	return nil
}

// cleanTags removes tags that are no longer used by any task. It returns
// nothing as its outcome has no effect on user transactions. Instead any
// useful information in can trasmit can be sent through logs.
func (s *protoServer) cleanTags(ctx context.Context) {
	slog.DebugContext(ctx, "Tag cleanup request received")
	// Context isn't used for the sql operation as it could expire.
	// However, knowing which endpoint/request_id triggered the cleanup could
	// prove useful
	if err := s.db.CleanTags(context.Background()); err != nil {
		// Don't return error as housekeeping issues shouln't block client requests
		slog.ErrorContext(ctx,
			"Failed to perform unused tags cleanup",
			logging.ErrKey, err,
		)
	}
}
