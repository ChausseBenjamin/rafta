package pb

import (
	"context"

	"github.com/ChausseBenjamin/rafta/internal/util"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *RaftaServer) NewUser(ctx context.Context, val *m.UserData) (*m.User, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "NewUser")
	return nil, nil
}

func (s *RaftaServer) UpdateUserInfo(ctx context.Context, val *m.User) (*emptypb.Empty, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "UpdateUserInfo")
	return nil, nil
}

func (s *RaftaServer) DeleteUser(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "DeleteUser")
	return nil, nil
}

func (s *RaftaServer) DeleteTask(ctx context.Context, val *m.UUID) (*emptypb.Empty, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "DeleteTask")
	return nil, nil
}

func (s *RaftaServer) CreateTask(ctx context.Context, val *m.TaskData) (*m.Task, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "CreateTask")
	return nil, nil
}
