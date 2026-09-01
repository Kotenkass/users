package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewLoggerReadsLogLevelFromEnv(t *testing.T) {
	t.Setenv(EnvLogLevel, "error")

	l := NewLogger()
	if got := l.GetLevel(); got != logrus.ErrorLevel {
		t.Fatalf("expected error level, got %s", got)
	}
}

func TestNewLoggerJSONFields(t *testing.T) {
	t.Setenv(EnvLogLevel, "debug")

	var buf bytes.Buffer
	l := NewLogger()
	l.SetOutput(&buf)

	Info(l, "user lookup completed", logrus.Fields{
		"request_id": "req-123",
		"user_id":    42,
	})

	got := decodeJSON(t, buf.String())
	if got["time"] == nil {
		t.Fatal("expected time field")
	}
	if got["level"] != "info" {
		t.Fatalf("expected level info, got %v", got["level"])
	}
	if got["service_name"] != ServiceName {
		t.Fatalf("expected service_name %q, got %v", ServiceName, got["service_name"])
	}
	if got["request_id"] != "req-123" {
		t.Fatalf("expected request_id req-123, got %v", got["request_id"])
	}
	if got["user_id"].(float64) != 42 {
		t.Fatalf("expected user_id 42, got %v", got["user_id"])
	}
}

func TestErrorIncludesErrorField(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger()
	l.SetOutput(&buf)

	err := errors.New("postgres connection refused")
	Error(l, err, "failed to connect to postgres", logrus.Fields{"retry": 2})

	got := decodeJSON(t, buf.String())
	if got["level"] != "error" {
		t.Fatalf("expected level error, got %v", got["level"])
	}
	if got["error"] != "postgres connection refused" {
		t.Fatalf("expected error field, got %v", got["error"])
	}
	if got["retry"].(float64) != 2 {
		t.Fatalf("expected retry 2, got %v", got["retry"])
	}
}

func TestEventBusinessLogging(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger()
	l.SetOutput(&buf)

	Event(l, "user.created", logrus.Fields{"user_id": 42})

	got := decodeJSON(t, buf.String())
	if got["event"] != "user.created" {
		t.Fatalf("expected event user.created, got %v", got["event"])
	}
	if got["msg"] != "business_event" {
		t.Fatalf("expected business_event message, got %v", got["msg"])
	}
	if got["user_id"].(float64) != 42 {
		t.Fatalf("expected user_id 42, got %v", got["user_id"])
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		value string
		want  logrus.Level
	}{
		{value: "debug", want: logrus.DebugLevel},
		{value: "DEBUG", want: logrus.DebugLevel},
		{value: "info", want: logrus.InfoLevel},
		{value: "", want: logrus.InfoLevel},
		{value: "error", want: logrus.ErrorLevel},
		{value: "invalid", want: logrus.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.value), func(t *testing.T) {
			if got := ParseLevel(tt.value); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func decodeJSON(t *testing.T, raw string) map[string]any {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("decode log output: %v\n%s", err, raw)
	}

	return got
}
