package pb

import (
	"context"

	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *raftaServer) GetTask(ctx context.Context, id *m.UUID) (*m.Task, error) {
	return nil, status.Error(codes.Unimplemented, "Server still under construction...")
}
