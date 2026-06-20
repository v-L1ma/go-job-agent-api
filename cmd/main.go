package main

import (
	"fmt"
	"job-agent-api/internal/database"
	"job-agent-api/internal/routes"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const (
	HOST     = "localhost"
	PORT     = 5432
	USER     = "jacob"
	PASSWORD = "password"
	DBNAME   = "bookstoreDB"
)

func main() {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		USER, PASSWORD, HOST, PORT, DBNAME)

	DB := database.NewDatabase(connString)
	defer DB.Pool.Close()

	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	routes.RegisterRoutes(e, DB)

	if err := e.Start(":1323"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}