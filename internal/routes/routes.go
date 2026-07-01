package routes

import (
	"job-agent-api/internal/database"
	"job-agent-api/internal/middlewares"

	"job-agent-api/internal/handlers"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

type CustomValidator struct {
    validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
    return cv.validator.Struct(i)
}

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

	e.Validator = &CustomValidator{
		validator: validator.New(),
	}

	api := e.Group("/api/v1")

	api.POST("/login", func(c *echo.Context) error {
		return handlers.Login(c, db)
	})

	api.POST("/register", func(c *echo.Context) error {
		return handlers.Register(c, db)
	})

	private := api.Group("")
	private.Use(middlewares.JWTMiddleware)

	private.GET("/jobs", func(c *echo.Context) error {
		return handlers.GetJobs(c, db)
	})

	private.GET("/jobs/:jobId", func(c *echo.Context) error{
		return handlers.GetJobById(c, db)
	})

	private.POST("/jobs/:jobId/rate", func (c *echo.Context) error{
		return handlers.RateJob(c, db)
	})

	private.POST("/users/preferences",func (c *echo.Context) error{
		return handlers.SetUserPreferences(c, db)
	})

	private.GET("/users/preferences", func (c *echo.Context) error{
		return handlers.GetUserPreferences(c, db)
	})

	private.POST("/users/cv", func (c *echo.Context) error{
		return handlers.UploadCv(c, db)
	})

	private.POST("/jobs/:jobId/cv", func (c *echo.Context) error{
		return handlers.GenerateCv(c, db)
	})

	private.GET("/users/cv", func (c *echo.Context) error{
		return handlers.GetUserCv(c, db)
	})
}