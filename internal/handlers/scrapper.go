package handlers

import (
	"job-agent-api/internal/database"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

func GetScrapperLogs(c *echo.Context, db *database.Database) error {
	// Implementation for getting scrapper logs
	id := c.Param("searchQueryId")
	var searchQueryId pgtype.UUID
	if err := searchQueryId.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	logs, err :=db.Query.GetSearchQueryScrapperLogs(c.Request().Context(), searchQueryId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Scrapper logs listados com sucesso", "logs": logs})
}

func GetScrapperExecutionJobs(c *echo.Context, db *database.Database) error {
	id := c.Param("executionId")
	var executionId pgtype.UUID
	if err := executionId.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	jobs, err :=db.Query.GetExecutionJobs(c.Request().Context(), executionId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Scrapper jobs listados com sucesso", "jobs": jobs})
}