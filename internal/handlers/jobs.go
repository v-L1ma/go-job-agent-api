package handlers

import (
	"job-agent-api/internal/database"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

func GetJobs(c *echo.Context, db *database.Database) error {
	var userID pgtype.UUID
	if err := userID.Scan("00000000-0000-0000-0000-000000000001"); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	jobs, err := db.Query.GetJobs(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, jobs)
}