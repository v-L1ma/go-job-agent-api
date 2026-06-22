package handlers

import (
	"job-agent-api/internal/database"
	"net/http"

	"github.com/labstack/echo/v5"
)

func GetJobs(c *echo.Context, db *database.Database) error {
	jobs, err := db.Query.GetJobs(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, jobs)
}