package logger_test

import (
	"errors"
	"io"

	"github.com/KOTENKASS/users/internal/logger"
	"github.com/sirupsen/logrus"
)

func ExampleInfo_additionalFields() {
	l := logger.NewLogger()
	l.SetOutput(io.Discard)

	logger.Info(l, "user lookup completed", logrus.Fields{
		"request_id": "example-request-id",
		"user_id":    42,
		"source":     "api",
	})
}

func ExampleDebug_diagnosticFields() {
	l := logger.NewLogger()
	l.SetOutput(io.Discard)

	logger.Debug(l, "database query finished", logrus.Fields{
		"query":       "select * from users where id = $1",
		"duration_ms": 12,
	})
}

func ExampleError_errorLogging() {
	l := logger.NewLogger()
	l.SetOutput(io.Discard)

	err := errors.New("postgres connection refused")
	logger.Error(l, err, "failed to connect to postgres", logrus.Fields{
		"request_id": "example-request-id",
		"retry":      2,
	})
}

func ExampleEvent_businessEventLogging() {
	l := logger.NewLogger()
	l.SetOutput(io.Discard)

	logger.Event(l, "user.created", logrus.Fields{
		"request_id": "example-request-id",
		"user_id":    42,
	})
}
