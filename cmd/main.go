package main

import (
	"fmt"
	"job-agent-api/internal/database"
	"job-agent-api/internal/routes"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-co-op/gocron/v2"
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

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		panic(err)
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(2*time.Hour),
		gocron.NewTask(func() {
			log.Println("Ativando Scrapper")
			var scrapperURL = getEnv("SCRAPPER_URL", "")
			if scrapperURL == "" {
				log.Println("SCRAPPER_URL não definido")
				return
			}
			resp, err := http.Get(scrapperURL)
			if err != nil {
				log.Println("Erro:", err)
				return
			}
			resp.Body.Close()

			log.Println("Ping enviado:", resp.StatusCode)
		}),
	)

	if err != nil {
		panic(err)
	}

	scheduler.Start()

	host := getEnv("DB_HOST", "localhost")
	portStr := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "jacob")
	password := getEnv("DB_PASSWORD", "password")
	dbname := getEnv("DB_NAME", "bookstoreDB")
	sslmode := getEnv("DB_SSLMODE", "disable")

	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 5432
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)

	DB := database.NewDatabase(connString)
	defer DB.Pool.Close()

	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))

	routes.RegisterRoutes(e, DB)

	if err := e.Start(":1323"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}