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

	api.POST("/refresh-token", func(c *echo.Context) error {
		return handlers.RefreshToken(c, db)
	})

	api.POST("/forgot-password", func(c *echo.Context) error {
		return handlers.ForgotPassword(c, db)
	})

	api.POST("/reset-password", func(c *echo.Context) error {
		return handlers.ResetPassword(c, db)
	})

	api.POST("/sync", func(c *echo.Context) error {
		return handlers.SyncJobsEmbeddings(c, db)
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

	private.POST("/jobs/:jobId/cv", func (c *echo.Context) error{
		return handlers.GenerateCv(c, db)
	})

	private.POST("/preferences",func (c *echo.Context) error{
		return handlers.SetUserPreferences(c, db)
	})

	private.GET("/preferences", func (c *echo.Context) error{
		return handlers.GetUserPreferences(c, db)
	})

	private.POST("/cv", func (c *echo.Context) error{
		return handlers.UploadCv(c, db)
	})

	private.GET("/cv", func (c *echo.Context) error{
		return handlers.GetUserCv(c, db)
	})

	private.GET("/cv/:cvId", func (c *echo.Context) error{
		return handlers.GetCvById(c, db)
	})

	private.GET("/cv/generated", func (c *echo.Context) error{
		return handlers.GetGeneratedCvs(c, db)
	})

	private.GET("/profile", func(c *echo.Context) error {
		return handlers.GetUserProfile(c, db)
	})

	private.PUT("/profile", func(c *echo.Context) error {
		return handlers.UpdateProfile(c, db)
	})

	private.PUT("/change-password", func(c *echo.Context) error {
		return handlers.ChangePassword(c, db)
	})

	private.GET("/statistics", func(c *echo.Context) error {
		return handlers.GetUserStatistics(c, db)
	})

	private.GET("/questions", func (c *echo.Context) error {
		return handlers.GetQuestions(c, db)
	})

	private.PUT("/questions/:questionId", func (c *echo.Context) error {
		return handlers.EditQuestion(c, db)
	})

	private.POST("/questions", func (c *echo.Context) error{
		return handlers.AnswerQuestion(c, db)
	})

	private.POST("/applications", func (c *echo.Context) error{
		return handlers.ApplyToJob(c, db)
	})

	private.GET("/applications", func (c *echo.Context) error{
		return handlers.GetApplications(c, db)
	})
	
	private.POST("/applications/:applicationId/rate", func (c *echo.Context) error{
		return handlers.RateApplication(c, db)
	})
}