package logging

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/util"
	"github.com/charmbracelet/log"
)

const (
	ErrKey = "error_message"
)

var (
	ErrInvalidLevel  = errors.New("invalid log level")
	ErrInvalidFormat = errors.New("invalid log format")
)

func Setup(lvlStr, fmtStr, outStr string) error {
	output, outputErr := setOutput(outStr)
	format, formatErr := setFormat(fmtStr)
	level, levelErr := setLevel(lvlStr)

	prefixStr := ""
	if format != log.JSONFormatter {
		prefixStr = "Rafta 🚢"
	}

	var h slog.Handler = log.NewWithOptions(
		output,
		log.Options{
			TimeFormat:   time.DateTime,
			Prefix:       prefixStr,
			Level:        level,
			ReportCaller: true,
			Formatter:    format,
		},
	)

	h = withTrackedContext(h, util.ReqIDKey, "request_id")
	h = withTrackedContext(h, util.ProtoMethodKey, "proto_method")
	h = withStackTrace(h)
	slog.SetDefault(slog.New(h))
	return errors.Join(outputErr, formatErr, levelErr)
}

func setLevel(target string) (log.Level, error) {
	for _, l := range []struct {
		prefix string
		level  log.Level
	}{
		{"deb", log.DebugLevel},
		{"inf", log.InfoLevel},
		{"warn", log.WarnLevel},
		{"err", log.ErrorLevel},
	} {
		if strings.HasPrefix(strings.ToLower(target), l.prefix) {
			return l.level, nil
		}
	}
	return log.InfoLevel, ErrInvalidLevel
}

func setFormat(f string) (log.Formatter, error) {
	switch f {
	case "plain", "text":
		return log.TextFormatter, nil
	case "json", "structured":
		return log.JSONFormatter, nil
	}
	return log.TextFormatter, ErrInvalidFormat
}

func setOutput(path string) (io.Writer, error) {
	switch path {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return os.Stdout, err
		} else {
			return f, nil
		}
	}
}
