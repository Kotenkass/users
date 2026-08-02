package main

import (
	"os"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"github.com/KOTENKASS/users/actions"
	dbhandler "github.com/KOTENKASS/users/lib"
)

func main() {
	e := echo.New()

	e.HideBanner = true
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())

	db := &dbhandler.DBHandler{}
	if err := db.ConnectPg(); err != nil {
		e.Logger.Fatal(err)
	}
	defer db.Close()

	if err := db.RunMigrations(); err != nil {
		e.Logger.Fatal(err)
	}

	actions.RegisterRoutes(e)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	e.Logger.Fatal(e.Start(":" + port))
}
