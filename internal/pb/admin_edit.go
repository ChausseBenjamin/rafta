package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/db"
	"github.com/ChausseBenjamin/rafta/internal/logging"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *AdminServer) DeleteUser(ctx context.Context, id *m.UUID) (*emptypb.Empty, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		return nil, err
	}
	if !hasRequiredRole(creds.Roles, allowedAdminRoles) {
		return nil, status.Error(
			codes.PermissionDenied,
			"User does not have the authority to delete another users",
		)
	}

	cmd := s.store.Common[db.DeleteUser]
	resp, err := cmd.ExecContext(ctx, id.Value)
	if err != nil {
		slog.ErrorContext(ctx, "Admin request to delete user failed",
			logging.ErrKey, err,
			db.RespMsgKey, resp,
		)
		return nil, err
	}
	if i, err := resp.RowsAffected(); i == 0 && err == nil {
		return nil, status.Errorf(codes.NotFound,
			"User %s does not exist in the database", id.Value,
		)
	}
	slog.InfoContext(ctx, "deleted user", db.RespMsgKey, resp)
	return &emptypb.Empty{}, nil
}

func (s *AdminServer) UpdateUser(ctx context.Context, val *m.User) (*emptypb.Empty, error) {
	creds, err := getCreds(ctx)
	if err != nil {
		return nil, err
	}
	if !hasRequiredRole(creds.Roles, allowedAdminRoles) {
		return nil, status.Error(
			codes.PermissionDenied,
			"User does not have the authority to update another users",
		)
	}

	return nil, status.Error(codes.Unimplemented, "Still under construction")
}
