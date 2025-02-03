package logging

import (
	"context"
	"log/slog"
)

type ctxTracker struct {
	ctxKey interface{}
	logKey string
	next   slog.Handler
}

func (h ctxTracker) Handle(ctx context.Context, r slog.Record) error {
	if v := ctx.Value(h.ctxKey); v != nil {
		r.AddAttrs(slog.Any(h.logKey, v))
	}
	return h.next.Handle(ctx, r)
}

func (h ctxTracker) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.next.Enabled(ctx, lvl)
}

func (h ctxTracker) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.next.WithAttrs(attrs)
}

func (h ctxTracker) WithGroup(name string) slog.Handler {
	return h.next.WithGroup(name)
}

func withTrackedContext(current slog.Handler, ctxKey interface{}, logKey string) *ctxTracker {
	return &ctxTracker{
		ctxKey: ctxKey,
		logKey: logKey,
		next:   current,
	}
}
