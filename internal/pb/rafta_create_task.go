package pb

import (
	"context"

	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *raftaServer) CreateTask(ctx context.Context, val *m.TaskData) (*m.Task, error) {
	return nil, status.Error(codes.Unimplemented, "Server still under construction...")
}
