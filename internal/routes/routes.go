package routes

import (
	"net/http"

	"job-agent-api/internal/database"

	"github.com/labstack/echo/v5"
	"job-agent-api/internal/handlers"
)

func RegisterRoutes(e *echo.Echo, db *database.Database) {
	e.GET("/", func(c *echo.Context) error {

		// var new = sqlc.CreateAuthorParams{
		// 	Name: "Jacob",
		// 	Cpf:  "12345678901",
		// };
		
		// db.Query.CreateAuthor(c.Request().Context(), new)
		authors, err := db.Query.ListAuthors(c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, authors)
	})

	e.GET("/jobs", func(c *echo.Context) error {
		return handlers.GetJobs(c, db)
	})
}