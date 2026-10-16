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

	api.POST("/login/google", func(c *echo.Context) error {
		return handlers.LoginGoogle(c, db)
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

	api.POST("/verify-email", func(c *echo.Context) error {
		return handlers.VerifyEmail(c, db)
	})

	api.POST("/resend-confirmation", func(c *echo.Context) error {
		return handlers.ResendConfirmation(c, db)
	})

	api.POST("/sync", func(c *echo.Context) error {
		return handlers.SyncJobsEmbeddings(c, db)
	})

	api.POST("/webhook/stripe", func(c *echo.Context) error {
		return handlers.StripeWebhook(c, db)
	})

	api.GET("/plans", func(c *echo.Context) error {
		return handlers.GetPlans(c, db)
	})

	private := api.Group("")
	private.Use(middlewares.JWTMiddleware)

	private.POST("/checkout/session/:planId", func(c *echo.Context) error {
		return handlers.CreatePaymentSession(c, db)
	})

	private.GET("/subscriptions/me", func(c *echo.Context) error {
		return handlers.GetMySubscription(c, db)
	})

	// Gestão da assinatura do próprio usuário (ownership sempre pelo JWT).
	private.POST("/subscriptions/me/cancel", func(c *echo.Context) error {
		return handlers.CancelMySubscription(c, db)
	})

	private.POST("/subscriptions/me/resume", func(c *echo.Context) error {
		return handlers.ResumeMySubscription(c, db)
	})

	private.POST("/subscriptions/me/portal-session", func(c *echo.Context) error {
		return handlers.CreateBillingPortalSession(c, db)
	})

	private.GET("/subscriptions/me/invoices", func(c *echo.Context) error {
		return handlers.GetMyInvoices(c, db)
	})

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

	private.GET("/questions", func(c *echo.Context) error {
		return handlers.GetQuestions(c, db)
	})

	private.PUT("/questions/:questionId", func(c *echo.Context) error {
		return handlers.EditQuestion(c, db)
	})

	private.POST("/questions", func(c *echo.Context) error {
		return handlers.AnswerQuestion(c, db)
	})

	private.POST("/applications", func(c *echo.Context) error {
		return handlers.ApplyToJob(c, db)
	})

	private.GET("/applications", func(c *echo.Context) error {
		return handlers.GetApplications(c, db)
	})

	private.POST("/applications/:applicationId/rate", func(c *echo.Context) error {
		return handlers.RateApplication(c, db)
	})

	admin := private.Group("/admin")
	admin.Use(middlewares.AdminMiddleware)

	admin.GET("/scrapper/logs/:searchQueryId", func(c *echo.Context) error {
		return handlers.GetScrapperLogs(c, db)
	})

	admin.GET("/scrapper/jobs/:executionId", func(c *echo.Context) error {
		return handlers.GetScrapperExecutionJobs(c, db)
	})

	admin.GET("/search-queries", func(c *echo.Context) error {
		return handlers.GetSearchQueries(c, db)
	})

	admin.GET("/roles", func(c *echo.Context) error {
		return handlers.GetRoles(c, db)
	})

	admin.GET("/roles/:id", func(c *echo.Context) error {
		return handlers.GetRoleById(c, db)
	})

	admin.POST("/roles", func(c *echo.Context) error {
		return handlers.CreateRole(c, db)
	})

	admin.PUT("/roles/:id", func(c *echo.Context) error {
		return handlers.UpdateRole(c, db)
	})

	admin.DELETE("/roles/:id", func(c *echo.Context) error {
		return handlers.DeleteRole(c, db)
	})

	admin.GET("/plans", func(c *echo.Context) error {
		return handlers.GetAdminPlans(c, db)
	})

	admin.GET("/subscriptions", func(c *echo.Context) error {
		return handlers.GetSubscriptions(c, db)
	})

	admin.GET("/subscriptions/:subscriptionId", func(c *echo.Context) error {
		return handlers.GetSubscriptionDetails(c, db)
	})

	admin.GET("/users/:userId/subscriptions", func(c *echo.Context) error {
		return handlers.GetUserSubscriptions(c, db)
	})

}
