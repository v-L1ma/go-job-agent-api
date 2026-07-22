package handlers

import (
	"encoding/base64"
	"encoding/json"
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

type cursorData struct {
	S float64 `json:"s"`
	C string  `json:"c"`
	I string  `json:"i"`
}

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
		Score: 			float64(job.Similarity),
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
	var params sqlc.GetJobsParams
	params.UserId = userID
	params.Limit = limit

	cursorStr := c.QueryParam("cursor")
	if cursorStr != "" {
		data, err := base64.StdEncoding.DecodeString(cursorStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid cursor"})
		}
		var cur cursorData
		if err := json.Unmarshal(data, &cur); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid cursor"})
		}
		t, err := time.Parse(time.RFC3339, cur.C)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid cursor"})
		}
		var id pgtype.UUID
		if err := id.Scan(cur.I); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid cursor"})
		}
		params.CursorSimilarity = pgtype.Float8{Float64: cur.S, Valid: true}
		params.CursorCreatedAt = pgtype.Timestamptz{Time: t, Valid: true}
		params.CursorID = id
	}

	jobs, err := db.Query.GetJobs(c.Request().Context(), params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	response := dto.ListJobsResponse{
		Jobs: toJobListDTO(jobs),
	}

	if len(jobs) > 0 {
		last := jobs[len(jobs)-1]
		cur := cursorData{
			S: float64(last.Similarity),
			C: last.CreatedAt.Time.Format(time.RFC3339),
			I: last.Id.String(),
		}
		data, _ := json.Marshal(cur)
		response.NextCursor = base64.StdEncoding.EncodeToString(data)
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

	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
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
		CreatedBy: userID.String(),
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		LastModifiedBy: userID.String(),
		LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	err = db.Query.EvaluateJob(c.Request().Context(), rating)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Muito obrigado pela sua avaliação!"})
}

func SyncJobsEmbeddings(c *echo.Context, db *database.Database) error {
	jobs, err := db.Query.GetJobsWithoutEmbeddings(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	for i, job := range jobs {
		// Process each job to generate embeddings
		titleEmbedding, err := services.GenerateEmbeddings(job.Title, "gemini-embedding-2")
		if err != nil {
			fmt.Println("Erro ao gerar embedding do titulo com gemini-embedding-2: ", err)
			titleEmbedding, err = services.GenerateEmbeddings(job.Title, "gemini-embedding-001")
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Erro ao gerar embedding do titulo: " + err.Error()})
			}
		}

		var descriptionEmbedding interface{}

		if job.Description != "" {
			descriptionEmbedding, err = services.GenerateEmbeddings(job.Description, "gemini-embedding-2")
			if err != nil {
				fmt.Println("Erro ao gerar embedding da descrição com gemini-embedding-2: ", err)
				descriptionEmbedding, err = services.GenerateEmbeddings(job.Description, "gemini-embedding-001")
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Erro ao gerar embedding da descrição: " + err.Error()})
				}
			}
		}

		err = db.Query.AddEmbeddings(c.Request().Context(), sqlc.AddEmbeddingsParams{
			Id: job.Id,
			TitleEmbedding: titleEmbedding,
			DescriptionEmbedding: descriptionEmbedding,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Erro ao salvar embeddings: " + err.Error()})
		}

		fmt.Println("Embedding salvo para: ", job.Title)
		if i == 60 {
			time.Sleep(1 * 60)
			fmt.Println("Pausa de 1 minuto após processar 90 vagas para evitar sobrecarga.")
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Sync completed successfully!"})
}