package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *raftaServer) UpdateUserInfo(ctx context.Context, data *m.UserData) (*timestamppb.Timestamp, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	modified, err := s.updateUser(ctx, creds.Subject, data)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "success")
	return timestamppb.New(modified.AsTime().UTC()), nil
}
