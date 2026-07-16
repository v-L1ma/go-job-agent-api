package handlers

import (
	"fmt"
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

func toJobDTO(job sqlc.GetJobsRow) dto.Job {
	return dto.Job{
		Id:             job.Id.String(),
		PlataformJobId: job.PlataformJobId,
		Title:          job.Title,
		Description:    job.Description,
		Url:            job.Url,
		IsApplied:      job.IsApplied,
		Status:         job.Status,
		Active:         job.Active,
		CreatedBy:      job.CreatedBy,
		CreatedAt:      job.CreatedAt.Time.Format(time.RFC3339),
		LastModifiedBy: job.LastModifiedBy,
		LastModifiedAt: job.LastModifiedAt.Time.Format(time.RFC3339),
		Platform:       job.Platform,
		Company:        job.Company,
		Score: 			job.Similarity,
	}
}

func toJobListDTO(jobs []sqlc.GetJobsRow) []dto.Job {
	result := make([]dto.Job, len(jobs))
	for i, j := range jobs {
		result[i] = toJobDTO(j)
	}
	return result
}

func GetJobs(c *echo.Context, db *database.Database) error {
	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	limitStr := c.QueryParam("limit")
	if limitStr == "" {
		limitStr = "10"
	}
	limit, err := helpers.ParseInt32(limitStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid limit"})
	}
	cursorStr := c.QueryParam("cursor")
	var cursor pgtype.Timestamptz
	if cursorStr != "" {
		t, err := time.Parse(time.RFC3339, cursorStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid cursor"})
		}
		cursor = pgtype.Timestamptz{Time: t, Valid: true}
	}

	jobs, err := db.Query.GetJobs(c.Request().Context(), sqlc.GetJobsParams{
		UserId: userID,
		Limit:  limit,
		Cursor: cursor,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	last := jobs[len(jobs)-1]

	response := dto.ListJobsResponse{
		Jobs: toJobListDTO(jobs),
		NextCursor: string(last.CreatedAt.Time.Format(time.RFC3339)),
	}

	return c.JSON(http.StatusOK, response)
}

func GetJobById(c *echo.Context, db *database.Database) error {
	id := c.Param("jobId")
	var jobID pgtype.UUID
	if err := jobID.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	job, err := db.Query.GetJobById(c.Request().Context(), jobID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, job)
}

type RateJobRequest struct {
	UserId string `json:"userId"`
	Liked bool `json:"liked"`
	Feedback string `json:"feedback"`
}

func RateJob (c *echo.Context, db *database.Database) error{
	id := c.Param("jobId")
	fmt.Println("Job ID:", id)
	var jobID pgtype.UUID
	if err := jobID.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req RateJobRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	var userID pgtype.UUID
	if err := userID.Scan(req.UserId); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	alreadyRated, err := db.Query.ExistsJobEvaluation(c.Request().Context(), sqlc.ExistsJobEvaluationParams{
		UserId: userID,
		JobId:  jobID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if alreadyRated {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Você já avaliou esta vaga."})
	}

	rating := sqlc.EvaluateJobParams{
		UserId: userID,
		JobId: jobID,
		Liked: req.Liked,
		Feedback: pgtype.Text{String: req.Feedback, Valid: true},
		Active: true,
		CreatedBy: req.UserId,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		LastModifiedBy: req.UserId,
		LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	err = db.Query.EvaluateJob(c.Request().Context(), rating)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Muito obrigado pela sua avaliação!"})
}