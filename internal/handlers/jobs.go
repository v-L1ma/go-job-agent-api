package handlers

import (
	"encoding/json"
	"fmt"
	"job-agent-api/internal/database"
	"job-agent-api/internal/dto"
	"job-agent-api/internal/helpers"
	sqlc "job-agent-api/internal/queries"
	"job-agent-api/internal/services"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

func toJobDTO(job sqlc.Job) dto.Job {
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
	}
}

func toJobListDTO(jobs []sqlc.Job) []dto.Job {
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

func ApplyToJob(c *echo.Context, db *database.Database) error {
	id := c.Param("jobId")
	var jobID pgtype.UUID
	if err := jobID.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	err := db.Query.CreateApplication(c.Request().Context(), sqlc.CreateApplicationParams{
		UserId: userID,
		JobId: jobID,
		Status: "applied",
		CreatedBy: "system",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		LastModifiedBy: "system",
		LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message":"Aplicação concluída com sucesso!"})
}

func AnswerQuestion(c *echo.Context, db *database.Database) error {
	id := c.Param("jobId")
	var jobID pgtype.UUID
	if err := jobID.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req dto.QuestoesRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	questions := make([]string, len(req.Questoes))

	for i, question := range req.Questoes {
		questions[i] = question.Pergunta
	}
		
	dbAnswer, err := db.Query.FindQuestionAnswer(c.Request().Context(), questions)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Erro ao buscar respostas no banco",
		})
	}

	answerMap := make(map[string]bool)

	for _, answer := range dbAnswer {
		answerMap[answer.Question] = true
	}

	filtered := make([]dto.QuestaoPersonalizada, 0, len(req.Questoes))
	for _, q := range req.Questoes {
		if !answerMap[q.Pergunta] {
			filtered = append(filtered, q)
		}
	}

	if len(filtered) == 0 {
		respostasDTO := make([]dto.RespostaQuestao, len(dbAnswer))
		for i, a := range dbAnswer {
			respostasDTO[i] = dto.RespostaQuestao{
				Pergunta: a.Question,
				Resposta: a.Answer,
			}
		}
		return c.JSON(http.StatusOK, map[string]any{
			"message": "Respostas recebidas com sucesso!",
			"data":    respostasDTO,
		})
	}

	fmt.Println("RESPOSTAS ENCONTRADAS:", dbAnswer)
	fmt.Println("QUESTOES FILTRADAS:", filtered)

	userCv, err := db.Query.GetUserCv(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	questoesJSON, err := json.Marshal(filtered)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var sb strings.Builder

	sb.WriteString("Você é um especialista em recrutamento e análise de currículos.")
	sb.WriteString("Sua tarefa é responder as perguntas abaixo com base no currículo do candidato.")
	sb.WriteString("")

	sb.WriteString("Regras importantes:")
	sb.WriteString("- Retorne APENAS um JSON válido")
	sb.WriteString("- O JSON deve conter as respostas para cada pergunta, se a pergunta tiver opções de resposta, escolha a mais adequada com base no currículo do candidato")
	sb.WriteString("- Se a pergunta for aberta, forneça uma resposta detalhada e completa com base no currículo do candidato")
	sb.WriteString("- Se a pergunta for de múltipla escolha, forneça a resposta correta com base no currículo do candidato")
	sb.WriteString("- Se a pergunta for de sim ou não, forneça a resposta correta com base no currículo do candidato")
	sb.WriteString("- Se não houver informações suficientes no currículo do candidato para responder a pergunta, forneça a resposta mais adequada com base nas informações disponíveis")
	sb.WriteString("- Não adicione explicações, comentários ou qualquer outro texto fora do JSON")
	sb.WriteString("- Não use markdown")
	sb.WriteString("- Não diminua o conteúdo do currículo, seja o mais completo possível")
	sb.WriteString("")

	sb.WriteString("Currículo:")
	sb.WriteString("```")
	sb.WriteString(userCv.ExtractedText)
	sb.WriteString("```")

	sb.WriteString("Perguntas:")
	sb.WriteString("```")
	sb.WriteString(string(questoesJSON))
	sb.WriteString("```")
	
	rawResponse, err := services.GenerateResponse(sb.String())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error do gemini": err.Error()})
	}

	rawStr, ok := rawResponse.(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Resposta inválida da IA",
		})
	}

	respostas, err := helpers.ParseQuestionsResponse(rawStr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Erro ao analisar resposta da IA: " + err.Error(),
		})
	}

	tx, err := db.Begin(c.Request().Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(c.Request().Context())

	qtx := db.Query.WithTx(tx)

	for _, question := range respostas.Respostas {
		err := qtx.CreateQuestion(c.Request().Context(), sqlc.CreateQuestionParams{
			UserId: userID,
			JobId: jobID,
			Question: question.Pergunta,
			Answer: question.Resposta,
			Active: true,
			CreatedBy: userID.String(),
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			LastModifiedBy: userID.String(),
			LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return err
		}
	}

	tx.Commit(c.Request().Context())

	return c.JSON(http.StatusOK, map[string]any{
		"message": "Respostas recebidas com sucesso!",
		"data": respostas,
	})

}