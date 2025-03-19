package pb

import (
	"context"

	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *raftaServer) DeleteTask(ctx context.Context, val *m.UUID) (*emptypb.Empty, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		return nil, err
	}

	return nil, status.Error(codes.Unimplemented, "Server still under construction...")
}
