package metrics

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/KOTENKASS/users/db"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "users_api"

var (
	// requestsTotal is a cumulative counter. Use rate(users_api_http_requests_total[1m])
	// in Prometheus queries to get requests per second.
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests handled by the users API since application start.",
		},
		[]string{"method", "route", "status"},
	)

	usersTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "users_total",
			Help:      "Total number of users currently in the system.",
		},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(usersTotal)
}

// Middleware records request counts for application routes.
func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)

			if c.Request().URL.Path == "/metrics" {
				return err
			}

			requestsTotal.WithLabelValues(c.Request().Method, routePath(c), statusCode(c, err)).Inc()

			return err
		}
	}
}

// Handler returns the Prometheus metrics HTTP handler.
func Handler() echo.HandlerFunc {
	return echo.WrapHandler(promhttp.Handler())
}

// SetUsersTotal updates the total number of users in the system.
func SetUsersTotal(count int) {
	usersTotal.Set(float64(count))
}

// ObserveUsersTotal reads the current user count and updates the users_total gauge.
func ObserveUsersTotal(dbh *db.UsersDBHandler) error {
	count, err := dbh.CountUsers()
	if err != nil {
		return err
	}

	usersTotal.Set(float64(count))
	return nil
}

func routePath(c echo.Context) string {
	path := c.Path()
	if path != "" {
		return path
	}
	return "not_found"
}

func statusCode(c echo.Context, err error) string {
	if c.Response().Status != 0 {
		return strconv.Itoa(c.Response().Status)
	}
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return strconv.Itoa(httpErr.Code)
		}
		return strconv.Itoa(http.StatusInternalServerError)
	}
	return strconv.Itoa(http.StatusOK)
}
