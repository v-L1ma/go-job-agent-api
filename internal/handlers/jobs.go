package handlers

import (
	"fmt"
	"job-agent-api/internal/database"
	"job-agent-api/internal/database/sqlc"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

func GetJobs(c *echo.Context, db *database.Database) error {
	id := c.Param("userId")
	var userID pgtype.UUID
	if err := userID.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	jobs, err := db.Query.GetJobs(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, jobs)
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