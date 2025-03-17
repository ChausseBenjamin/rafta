package pb

import (
	"context"

	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *raftaServer) GetUserInfo(ctx context.Context, _ *emptypb.Empty) (*m.User, error) {
	return nil, status.Error(codes.Unimplemented, "Server still under construction...")
}
