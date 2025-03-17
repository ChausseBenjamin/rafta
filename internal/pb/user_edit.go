package pb

import (
	"context"

	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *raftaServer) NewUser(ctx context.Context, val *m.UserData) (*m.User, error) {
	return nil, status.Error(codes.Unimplemented, "Server still under construction...")
}

func (s *raftaServer) UpdateUserInfo(ctx context.Context, val *m.User) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Server still under construction...")
}

func (s *raftaServer) DeleteUser(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Server still under construction...")
}

func (s *raftaServer) DeleteTask(ctx context.Context, val *m.UUID) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Server still under construction...")
}

func (s *raftaServer) CreateTask(ctx context.Context, val *m.TaskData) (*m.Task, error) {
	return nil, status.Error(codes.Unimplemented, "Server still under construction...")
}
