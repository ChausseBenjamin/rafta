package server

import (
	"context"
	"log/slog"

	m "github.com/ChausseBenjamin/rafta/internal/server/model"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s Service) GetAllUsers(ctx context.Context, empty *emptypb.Empty) (*m.UserList, error) {
	slog.ErrorContext(ctx, "GetAllUsers not implemented yet")
	// TODO: implement GetAllUsers
	return nil, nil
}

func (s Service) GetUser(ctx context.Context, id *m.UserID) (*m.User, error) {
	slog.ErrorContext(ctx, "GetUser not implemented yet")
	// TODO: implement GetUser
	return nil, nil
}

func (s Service) DeleteUser(ctx context.Context, id *m.UserID) (*emptypb.Empty, error) {
	slog.ErrorContext(ctx, "DeleteUser not implemented yet")
	// TODO: implement DeleteUser
	return nil, nil
}

func (s Service) UpdateUser(ctx context.Context, u *m.User) (*m.User, error) {
	slog.ErrorContext(ctx, "UpdateUser not implemented yet")
	// TODO: implement UpdateUser
	return nil, nil
}

func (s Service) CreateUser(ctx context.Context, data *m.UserData) (*m.User, error) {
	slog.ErrorContext(ctx, "CreateUser not implemented yet")
	// TODO: implement CreateUser
	return nil, nil
}
