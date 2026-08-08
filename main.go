package main

import (
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"github.com/KOTENKASS/users/actions"
	"github.com/KOTENKASS/users/db"
)

func main() {
	e := echo.New()

	e.HideBanner = true
	if strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug") {
		e.Debug = true
	}
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())

	dbh := db.UsersDBHandler{}
	if err := dbh.ConnectPg(); err != nil {
		e.Logger.Fatal(err)
	}
	defer dbh.Close()

	if err := dbh.RunMigrations(); err != nil {
		e.Logger.Fatal(err)
	}

	actions.RegisterRoutes(e)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	e.Logger.Fatal(e.Start(":" + port))
}
