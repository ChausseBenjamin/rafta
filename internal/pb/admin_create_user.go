package pb

import (
	"context"

	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *adminServer) CreateUser(ctx context.Context, req *m.UserSignupRequest) (*emptypb.Empty, error) {
	_, err := s.newUser(ctx, req)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
