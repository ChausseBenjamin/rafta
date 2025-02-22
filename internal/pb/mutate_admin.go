package pb

import (
	"context"

	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *AdminServer) DeleteUser(ctx context.Context, id *m.UUID) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *AdminServer) UpdateUser(ctx context.Context, val *m.User) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *AdminServer) CreateUser(ctx context.Context, val *m.UserCreationMsg) (*m.User, error) {
	return nil, nil
}
