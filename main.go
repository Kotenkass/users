package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KOTENKASS/users/actions"
	"github.com/KOTENKASS/users/db"
	appLogger "github.com/KOTENKASS/users/internal/logger"
	appMetrics "github.com/KOTENKASS/users/internal/metrics"
	appMiddleware "github.com/KOTENKASS/users/internal/middleware"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

func main() {
	l := appLogger.NewLogger()

	e := echo.New()
	e.HideBanner = true
	if appLogger.ParseLevel(os.Getenv(appLogger.EnvLogLevel)) == logrus.DebugLevel {
		e.Debug = true
	}
	e.Use(appMiddleware.RequestID(l))
	e.Use(echoMiddleware.Recover())
	e.GET("/healthz", healthzHandler)

	dbh := db.UsersDBHandler{}
	if err := dbh.ConnectPg(); err != nil {
		l.Fatal(err)
	}
	defer dbh.Close()

	if err := dbh.RunMigrations(); err != nil {
		l.Fatal(err)
	}
	if err := appMetrics.ObserveUsersTotal(&dbh); err != nil {
		l.Fatal(err)
	}

	e.Use(appMetrics.Middleware())
	e.GET("/metrics", appMetrics.Handler())
	e.GET("/readyz", readyzHandler(&dbh))

	actions.RegisterRoutes(e)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	startHTTPServer(l, e, port)
	waitForShutdown(l, e)
}

func healthzHandler(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func readyzHandler(dbh *db.UsersDBHandler) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()

		if err := dbh.Ping(ctx); err != nil {
			return c.String(http.StatusServiceUnavailable, "database is not ready")
		}

		return c.String(http.StatusOK, "ok")
	}
}

func startHTTPServer(log *logrus.Logger, e *echo.Echo, port string) {
	go func() {
		addr := ":" + port
		log.WithField("addr", addr).Info("http server starting")
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("http server failed")
		}
	}()
}

func waitForShutdown(log *logrus.Logger, e *echo.Echo) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Info("shutdown signal received")
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("shutdown http server failed")
	}
}
