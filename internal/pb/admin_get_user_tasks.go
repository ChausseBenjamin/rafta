package pb

import (
	"context"

	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *adminServer) GetUserTasks(context.Context, *m.UUID) (*m.TaskList, error) {
	return nil, status.Error(codes.Unimplemented, "Server under construction still...")
}
