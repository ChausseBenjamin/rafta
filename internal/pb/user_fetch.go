package pb

import (
	"context"

	"github.com/ChausseBenjamin/rafta/internal/util"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *raftaServer) GetTask(ctx context.Context, id *m.UUID) (*m.Task, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "GetTask")
	return nil, nil
}

func (s *raftaServer) GetUserInfo(ctx context.Context, _ *emptypb.Empty) (*m.User, error) {
	ctx = context.WithValue(ctx, util.ProtoMethodKey, "GetUserInfo")
	return nil, nil
}
