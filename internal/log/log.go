package log

import (
	"log/slog"
	"math"
	"os"
)

// Init configures the default slog logger. When debug is true, logs are
// written to stderr at Debug level. Otherwise logging is effectively silent.
func Init(debug bool) {
	level := slog.Level(math.MaxInt) // silent
	if debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}
