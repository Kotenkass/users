package main

import (
	"os"

	"github.com/KOTENKASS/users/actions"
	"github.com/KOTENKASS/users/db"
	appLogger "github.com/KOTENKASS/users/internal/logger"
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

	dbh := db.UsersDBHandler{}
	if err := dbh.ConnectPg(); err != nil {
		l.Fatal(err)
	}
	defer dbh.Close()

	if err := dbh.RunMigrations(); err != nil {
		l.Fatal(err)
	}

	actions.RegisterRoutes(e)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	l.Fatal(e.Start(":" + port))
}
