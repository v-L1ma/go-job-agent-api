package routes

import (

	"job-agent-api/internal/database"

	"github.com/labstack/echo/v5"
	"job-agent-api/internal/handlers"
)

func RegisterRoutes(e *echo.Echo, db *database.Database) {
	// e.GET("/", func(c *echo.Context) error {

	// 	// var new = sqlc.CreateAuthorParams{
	// 	// 	Name: "Jacob",
	// 	// 	Cpf:  "12345678901",
	// 	// };
		
	// 	// db.Query.CreateAuthor(c.Request().Context(), new)
	// 	authors, err := db.Query.ListAuthors(c.Request().Context())
	// 	if err != nil {
	// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	// 	}
	// 	return c.JSON(http.StatusOK, authors)
	// })

	e.GET("/jobs/me/:userId", func(c *echo.Context) error {
		return handlers.GetJobs(c, db)
	})

	e.GET("/jobs/:jobId", func(c *echo.Context) error{
		return handlers.GetJobById(c, db)
	})

	e.POST("/jobs/:jobId/rate", func (c *echo.Context) error{
		return handlers.RateJob(c, db)
	})

	e.POST("/users/preferences/:userId",func (c *echo.Context) error{
		return handlers.SetUserPreferences(c, db)
	})

	e.GET("users/preferences/:userId", func (c *echo.Context) error{
		return handlers.GetUserPreferences(c, db)
	})

	e.POST("users/cv/:userId", func (c *echo.Context) error{
		return handlers.UploadCv(c, db)
	})

	e.POST("jobs/:jobId/user/:userId", func (c *echo.Context) error{
		return handlers.GenerateCv(c, db)
	})
}