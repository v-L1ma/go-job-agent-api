package handlers

import (
	"job-agent-api/internal/database"
	"net/http"

	"github.com/labstack/echo/v5"
)

func GetSearchQueries(c *echo.Context, db *database.Database) error {
	logs, err :=db.Query.GetSearchQueries(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Search queries listados com sucesso", "logs": logs})
}