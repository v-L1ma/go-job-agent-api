package main

import (
	"fmt"
	"job-agent-api/internal/database"
	"job-agent-api/internal/routes"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	host := getEnv("DB_HOST", "localhost")
	portStr := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "jacob")
	password := getEnv("DB_PASSWORD", "password")
	dbname := getEnv("DB_NAME", "bookstoreDB")

	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 5432
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, host, port, dbname)

	DB := database.NewDatabase(connString)
	defer DB.Pool.Close()

	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	corsOrigins := getEnv("CORS_ORIGINS", "http://localhost:3000")
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: strings.Split(corsOrigins, ","),
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))

	routes.RegisterRoutes(e, DB)

	if err := e.Start(":1323"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}