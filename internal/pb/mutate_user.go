package pb

import (
	"context"

	"github.com/ChausseBenjamin/rafta/internal/util"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *UserServer) NewUser(ctx context.Context, val *m.UserData) (*m.User, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "NewUser")
	return nil, nil
}

func (s *UserServer) UpdateUserInfo(ctx context.Context, val *m.User) (*emptypb.Empty, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "UpdateUserInfo")
	return nil, nil
}

func (s *UserServer) DeleteUser(ctx context.Context, val *emptypb.Empty) (*emptypb.Empty, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "DeleteUser")
	return nil, nil
}

func (s *UserServer) DeleteTask(ctx context.Context, val *m.UUID) (*emptypb.Empty, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "DeleteTask")
	return nil, nil
}

func (s *UserServer) UpdateTask(ctx context.Context, val *m.TaskUpdate) (*emptypb.Empty, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "UpdateTask")
	return nil, nil
}

func (s *UserServer) CreateTask(ctx context.Context, val *m.TaskData) (*m.Task, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "CreateTask")
	return nil, nil
}
