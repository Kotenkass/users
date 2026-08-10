package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/KOTENKASS/users/internal/logger"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

const RequestIDContextKey = "request_id"

// RequestID adds a request ID to the context and response, then logs each request.
func RequestID(log *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := requestID(c)
			c.Set(RequestIDContextKey, requestID)
			c.Response().Header().Set(echo.HeaderXRequestID, requestID)

			start := time.Now()
			err := next(c)
			elapsed := time.Since(start)

			level := logrus.InfoLevel
			if err != nil || c.Response().Status >= http.StatusInternalServerError {
				level = logrus.ErrorLevel
			}

			logger.Log(log, level, "request completed", logrus.Fields{
				"request_id":  requestID,
				"method":      c.Request().Method,
				"path":        c.Request().URL.Path,
				"status":      c.Response().Status,
				"duration_ms": elapsed.Milliseconds(),
			})

			return err
		}
	}
}

func requestID(c echo.Context) string {
	requestID := c.Request().Header.Get(echo.HeaderXRequestID)
	if requestID != "" {
		return requestID
	}
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
