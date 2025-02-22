package intercept

import (
	"context"
	"log/slog"

	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/ChausseBenjamin/rafta/internal/util"
	"github.com/hashicorp/go-uuid"
	"google.golang.org/grpc"
)

// gRPC interceptor to tag requests with a unique identifier
func Tagging(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	id, err := uuid.GenerateUUID()
	if err != nil {
		slog.Error("Unable to generate UUID for request", logging.ErrKey, err)
	}
	ctx = context.WithValue(ctx, util.ReqIDKey, id)
	slog.DebugContext(ctx, "Tagging request with UUID", "value", id)

	return handler(ctx, req)
}
