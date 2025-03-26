package pb

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/auth"
	m "github.com/ChausseBenjamin/rafta/pkg/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *raftaServer) UpdateCredentials(ctx context.Context, req *m.PasswdMessage) (*timestamppb.Timestamp, error) {
	creds, err := auth.GetCreds(ctx, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	modified, err := s.updateUserCredentials(ctx, creds.UserID, req.Secret)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "success", "user_id", creds.UserID)
	return modified, nil
}
