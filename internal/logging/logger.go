// Package logging provides structured logging with slog for AGIS.
package logging

import (
"context"
"io"
"log/slog"
"os"
"strings"
"time"
)

// Level represents logging levels.
type Level = slog.Level

const (
LevelDebug = slog.LevelDebug
LevelInfo  = slog.LevelInfo
LevelWarn  = slog.LevelWarn
LevelError = slog.LevelError
)

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
*slog.Logger
level *slog.LevelVar
}

// Config holds logger configuration.
type Config struct {
Level      string
Format     string
Output     io.Writer
AddSource  bool
TimeFormat string
}

// DefaultConfig returns sensible defaults for production.
func DefaultConfig() Config {
return Config{
Level:      "info",
Format:     "json",
Output:     os.Stdout,
AddSource:  false,
TimeFormat: time.RFC3339,
}
}

// New creates a new Logger with the given configuration.
func New(cfg Config) *Logger {
level := new(slog.LevelVar)
level.Set(parseLevel(cfg.Level))

opts := &slog.HandlerOptions{
Level:     level,
AddSource: cfg.AddSource,
}

var handler slog.Handler
output := cfg.Output
if output == nil {
output = os.Stdout
}

if cfg.Format == "text" {
handler = slog.NewTextHandler(output, opts)
} else {
handler = slog.NewJSONHandler(output, opts)
}

return &Logger{
Logger: slog.New(handler),
level:  level,
}
}

// NewFromEnv creates a logger configured from environment variables.
func NewFromEnv() *Logger {
cfg := DefaultConfig()
if level := os.Getenv("LOG_LEVEL"); level != "" {
cfg.Level = level
}
if format := os.Getenv("LOG_FORMAT"); format != "" {
cfg.Format = format
}
return New(cfg)
}

// SetLevel dynamically changes the log level.
func (l *Logger) SetLevel(level string) {
l.level.Set(parseLevel(level))
}

// With returns a new Logger with the given attributes.
func (l *Logger) With(args ...any) *Logger {
return &Logger{
Logger: l.Logger.With(args...),
level:  l.level,
}
}

// WithComponent returns a logger tagged with a component name.
func (l *Logger) WithComponent(name string) *Logger {
return l.With("component", name)
}

// WithRequestID returns a logger tagged with a request ID.
func (l *Logger) WithRequestID(id string) *Logger {
return l.With("request_id", id)
}

// WithUser returns a logger tagged with user information.
func (l *Logger) WithUser(userID string) *Logger {
return l.With("user_id", userID)
}

// WithGuild returns a logger tagged with guild information.
func (l *Logger) WithGuild(guildID string) *Logger {
return l.With("guild_id", guildID)
}

// WithError returns a logger with error details.
func (l *Logger) WithError(err error) *Logger {
if err == nil {
return l
}
return l.With("error", err.Error())
}

// Fatal logs at error level and exits.
func (l *Logger) Fatal(msg string, args ...any) {
l.Error(msg, args...)
os.Exit(1)
}

func parseLevel(s string) slog.Level {
switch strings.ToLower(s) {
case "debug":
return slog.LevelDebug
case "info":
return slog.LevelInfo
case "warn", "warning":
return slog.LevelWarn
case "error":
return slog.LevelError
default:
return slog.LevelInfo
}
}

type contextKey string

const (
loggerKey    contextKey = "logger"
requestIDKey contextKey = "request_id"
)

// WithLogger adds a logger to the context.
func WithLogger(ctx context.Context, l *Logger) context.Context {
return context.WithValue(ctx, loggerKey, l)
}

// FromContext retrieves the logger from context.
func FromContext(ctx context.Context) *Logger {
if l, ok := ctx.Value(loggerKey).(*Logger); ok {
return l
}
return defaultLogger
}

// SetRequestID adds a request ID to the context.
func SetRequestID(ctx context.Context, id string) context.Context {
return context.WithValue(ctx, requestIDKey, id)
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
if id, ok := ctx.Value(requestIDKey).(string); ok {
return id
}
return ""
}

var defaultLogger = NewFromEnv()

// Default returns the default logger.
func Default() *Logger {
return defaultLogger
}

// SetDefault sets the default logger.
func SetDefault(l *Logger) {
defaultLogger = l
slog.SetDefault(l.Logger)
}

// Package-level convenience functions.
func Debug(msg string, args ...any) { defaultLogger.Debug(msg, args...) }
func Info(msg string, args ...any)  { defaultLogger.Info(msg, args...) }
func Warn(msg string, args ...any)  { defaultLogger.Warn(msg, args...) }
func Error(msg string, args ...any) { defaultLogger.Error(msg, args...) }
