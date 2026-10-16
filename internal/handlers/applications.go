package handlers

import (
	"errors"
	"job-agent-api/internal/database"
	"job-agent-api/internal/dto"
	"job-agent-api/internal/helpers"
	sqlc "job-agent-api/internal/queries"
	"job-agent-api/internal/services"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

func GetApplications(c *echo.Context, db *database.Database) error {
	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	err := userID.Scan(claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	existUser, err := db.Query.ExistsUserById(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if !existUser {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Usuário não encontrado."})
	}

	applications, err := db.Query.GetApplicationsWithQuestions(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	result := make([]dto.ApplicationWithQuestions, 0, len(applications))
	for _, app := range applications {
		questions := make([]dto.QuestionAnswer, 0)

		if questionsRaw, ok := app.Questions.([]interface{}); ok {
			for _, q := range questionsRaw {
				if qMap, ok := q.(map[string]interface{}); ok {
					qa := dto.QuestionAnswer{}
					if id, ok := qMap["id"].(string); ok {
						qa.Id = id
					}
					if question, ok := qMap["question"].(string); ok {
						qa.Question = question
					}
					if answer, ok := qMap["answer"].(string); ok {
						qa.Answer = answer
					}
					if t, ok := qMap["type"].(string); ok {
						qa.Type = t
					}
					if opts, ok := qMap["options"].([]interface{}); ok {
						for _, o := range opts {
							if s, ok := o.(string); ok {
								qa.Options = append(qa.Options, s)
							}
						}
					}
					questions = append(questions, qa)
				}
			}
		}

		result = append(result, dto.ApplicationWithQuestions{
			Id:             app.Id.String(),
			Title:          app.Title.String,
			Company:        app.Company.String,
			Platform:       app.Platform.String,
			Status:         app.Status,
			CreatedAt:      app.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			QuestionsCount: app.QuestionsCount,
			Questions:      questions,
		})
	}

	return c.JSON(http.StatusOK, dto.ResponseBase[[]dto.ApplicationWithQuestions]{
		Message: "Candidaturas encontradas com sucesso!",
		Data:    result,
	})
}

func ApplyToJob(c *echo.Context, db *database.Database) error {
	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req dto.ApplyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	var jobID pgtype.UUID
	if err := jobID.Scan(req.JobId); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Enforcement do limite diário de candidaturas do plano — antes do INSERT.
	if err := services.CheckDailyApplicationLimit(c.Request().Context(), db.Query, userID); err != nil {
		var limitErr *services.PlanLimitError
		if errors.As(err, &limitErr) {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": limitErr.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	err := db.Query.CreateApplication(c.Request().Context(), sqlc.CreateApplicationParams{
		UserId: userID,
		JobId: jobID,
		Status: req.Status,
		Obs: pgtype.Text{String: req.Observation, Valid: true},
		Platform: pgtype.Text{String: req.Platform, Valid: true},
		CreatedBy: "system",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		LastModifiedBy: "system",
		LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message":"Candidatura concluída com sucesso!"})
}

func RateApplication(c *echo.Context, db *database.Database) error {
	id := c.Param("applicationId")
	var applicationId pgtype.UUID
	if err := applicationId.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req dto.RateApplicationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": helpers.ValidationErrors(err),
		})
	}

	err := db.Query.CreateApplicationRate(c.Request().Context(), sqlc.CreateApplicationRateParams{
		UserId: userID,
		ApplicationId: applicationId,
		Liked: req.Liked,
		Feedback: pgtype.Text{String: req.Feedback, Valid: true},
		CreatedBy: "system",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		LastModifiedBy: "system",
		LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message":"Candidatura avaliada com sucesso!"})
}