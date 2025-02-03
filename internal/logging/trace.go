package logging

import (
	"context"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
)

const (
	prgCount = 20
	defSkip  = 6
)

type stackTracer struct {
	h     slog.Handler
	nSkip int
}

func (h stackTracer) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.h.Enabled(ctx, lvl)
}

func (h stackTracer) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.h.WithAttrs(attrs)
}

func (h stackTracer) WithGroup(name string) slog.Handler {
	return h.h.WithGroup(name)
}

func (h stackTracer) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < slog.LevelError {
		return h.h.Handle(ctx, r)
	}

	trace := h.GetTrace()
	r.AddAttrs(slog.String("trace", trace))

	return h.h.Handle(ctx, r)
}

func (h stackTracer) GetTrace() string {
	var b strings.Builder
	pc := make([]uintptr, prgCount)
	n := runtime.Callers(h.nSkip, pc)
	frames := runtime.CallersFrames(pc[:n])

	for frame, more := frames.Next(); more; frame, more = frames.Next() {
		b.WriteString(frame.Function + "\n    " + frame.File + ":" + strconv.Itoa(frame.Line) + "\n")
	}
	return b.String()
}

func withStackTrace(h slog.Handler) slog.Handler {
	return stackTracer{
		h:     h,
		nSkip: defSkip,
	}
}
