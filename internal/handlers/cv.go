package handlers

import (
	"fmt"
	"io"
	"job-agent-api/internal/database"
	"job-agent-api/internal/database/sqlc"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/ledongthuc/pdf"
)

type EvaluateCVRequest struct {
	UserId string `json:"userId"`
	Liked bool `json:"liked"`
	Feedback string `json:"feedback"`
}

func evaluateCV(c *echo.Context, db *database.Database) error {
	id := c.Param("cvId")
	var cvId pgtype.UUID
	if err := cvId.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req EvaluateCVRequest
	if err := c.Bind(&req); err != nil{
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	var userID pgtype.UUID
	if err := userID.Scan(req.UserId); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	err := db.Query.EvaluateCv(c.Request().Context(), sqlc.EvaluateCvParams{
		UserId: userID,
		GeneratedCvId: cvId,
		Liked: req.Liked,
		Feedback: pgtype.Text{String: req.Feedback, Valid: true},
		Active: true,
		CreatedBy: req.UserId,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		LastModifiedBy: req.UserId,
		LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}) 

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Muito obrigado pela sua avaliação!"})
}

func UploadCv(c *echo.Context, db *database.Database) error {
	// Obtém o arquivo enviado no campo "cv"
	fileHeader, err := c.FormFile("cv")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Currículo não enviado",
		})
	}

	// Abre o arquivo enviado
	src, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Erro ao abrir o arquivo",
		})
	}
	defer src.Close()

	// Cria um arquivo temporário
	tmpFile, err := os.CreateTemp("", "*.pdf")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Erro ao criar arquivo temporário",
		})
	}
	defer os.Remove(tmpFile.Name()) // Remove o arquivo ao final
	defer tmpFile.Close()

	// Copia o upload para o arquivo temporário
	_, err = io.Copy(tmpFile, src)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Erro ao salvar arquivo temporário",
		})
	}

	// Faz a leitura do PDF
	pdf.DebugOn = true

	f, reader, err := pdf.Open(tmpFile.Name())
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Erro ao abrir o PDF",
		})
	}
	defer f.Close()

	textReader, err := reader.GetPlainText()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Erro ao extrair texto do PDF",
		})
	}

	textBytes, err := io.ReadAll(textReader)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Erro ao ler conteúdo do PDF",
		})
	}

	content := string(textBytes)

	fmt.Println(content)

	return c.JSON(http.StatusOK, map[string]any{
		"filename": fileHeader.Filename,
		"size":     fileHeader.Size,
		"type":     fileHeader.Header.Get("Content-Type"),
		"content":  content,
	})
}