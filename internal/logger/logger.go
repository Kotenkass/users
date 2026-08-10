package logger

import (
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// EnvLogLevel is the environment variable used to configure the minimum log level.
	EnvLogLevel = "LOG_LEVEL"

	// ServiceName is emitted on every structured log entry from this service.
	ServiceName = "users"
)

// NewLogger returns a structured JSON logger configured from LOG_LEVEL.
//
// Supported LOG_LEVEL values are debug, info, and error. Unknown or empty
// values fall back to info. Every emitted JSON entry includes:
//   - time: RFC3339Nano timestamp from the log entry
//   - level: logrus level string, for example "info", "debug", or "error"
//   - service_name: this service name
func NewLogger() *logrus.Logger {
	l := logrus.New()
	l.SetFormatter(&serviceJSONFormatter{
		service: ServiceName,
		formatter: &logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		},
	})
	l.SetOutput(os.Stdout)
	l.SetLevel(ParseLevel(os.Getenv(EnvLogLevel)))

	return l
}

// ParseLevel converts a user-facing log level into logrus.Level.
//
// Supported values are "debug", "info", and "error". Values are trimmed and
// case-insensitive. Unknown or empty values fall back to info.
func ParseLevel(value string) logrus.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return logrus.DebugLevel
	case "error":
		return logrus.ErrorLevel
	case "info", "":
		return logrus.InfoLevel
	default:
		return logrus.InfoLevel
	}
}

// NewEntry returns an entry with service_name already set.
//
// Use the returned entry's Debug, Info, Error, WithField, WithFields, and
// WithError methods to set the entry level and add request-specific fields.
func NewEntry(log *logrus.Logger) *logrus.Entry {
	if log == nil {
		log = NewLogger()
	}

	return log.WithField("service_name", ServiceName)
}

// Log writes msg at level with additional fields.
//
// This helper is useful when the level is selected dynamically. For static
// levels, prefer Info, Debug, Error, or an entry's methods directly.
func Log(log *logrus.Logger, level logrus.Level, msg string, fields logrus.Fields) {
	entry := NewEntry(log)
	if len(fields) > 0 {
		entry = entry.WithFields(fields)
	}

	switch level {
	case logrus.DebugLevel:
		entry.Debug(msg)
	case logrus.ErrorLevel:
		entry.Error(msg)
	default:
		entry.Info(msg)
	}
}

// Info writes an info-level business or operational log entry.
func Info(log *logrus.Logger, msg string, fields logrus.Fields) {
	Log(log, logrus.InfoLevel, msg, fields)
}

// Debug writes a debug-level diagnostic log entry.
func Debug(log *logrus.Logger, msg string, fields logrus.Fields) {
	Log(log, logrus.DebugLevel, msg, fields)
}

// Error writes an error-level log entry and attaches err when provided.
func Error(log *logrus.Logger, err error, msg string, fields logrus.Fields) {
	entry := NewEntry(log)
	if len(fields) > 0 {
		entry = entry.WithFields(fields)
	}
	if err != nil {
		entry = entry.WithError(err)
	}

	entry.Error(msg)
}

// Event writes an info-level business event. The event name is emitted in the
// "event" field, and fields are merged into the structured log entry.
func Event(log *logrus.Logger, eventName string, fields logrus.Fields) {
	merged := logrus.Fields{"event": eventName}
	for k, v := range fields {
		merged[k] = v
	}

	Info(log, "business_event", merged)
}

type serviceJSONFormatter struct {
	service   string
	formatter logrus.Formatter
}

func (f *serviceJSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	fields := make(logrus.Fields, len(entry.Data)+1)
	for k, v := range entry.Data {
		fields[k] = v
	}
	if _, ok := fields["service_name"]; !ok {
		fields["service_name"] = f.service
	}

	cloned := *entry
	cloned.Data = fields

	return f.formatter.Format(&cloned)
}
