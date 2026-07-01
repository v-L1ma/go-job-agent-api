package handlers

import (
	"job-agent-api/internal/database"
	"job-agent-api/internal/database/sqlc"
	"job-agent-api/internal/dto"
	"job-agent-api/internal/services"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

type SetUserPreferencesRequest struct {
	Skills []string `json:"skills"`
	Levels []string `json:"levels"`
}

func SetUserPreferences(c *echo.Context, db *database.Database) error {
	id := c.Get("user").(*services.Claims)
	
	var cvId pgtype.UUID
	if err := cvId.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req SetUserPreferencesRequest
	if err := c.Bind(&req); err != nil{
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	var userID pgtype.UUID
	if err := userID.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	existUser, err := db.Query.ExistsUserById(c.Request().Context(), userID)
	if err != nil{
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if !existUser {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Usuário não encontrado."})
	}

	if len(req.Skills) <= 0 && len(req.Levels) <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Informe ao menos uma Habilidade e uma senioridade."}) 
	}

	alreadyCreated, err := db.Query.FindUserPreferences(c.Request().Context(), userID)

	if err != nil{
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if alreadyCreated {
		err = db.Query.UpdateUserPreferences(c.Request().Context(), sqlc.UpdateUserPreferencesParams{
			UserId: userID,
			Skills: req.Skills,
			Levels: req.Levels,
			Active: true,
			LastModifiedBy: userID.String(),
			LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})

		if err != nil{
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Preferências atualizadas com sucesso!"})
	}

	err = db.Query.CreateUserPreferences(c.Request().Context(), sqlc.CreateUserPreferencesParams{
		UserId: userID,
		Skills: req.Skills,
		Levels: req.Levels,
		Active: true,
		LastModifiedBy: userID.String(),
		LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		CreatedBy: userID.String(),
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})

	if err != nil{
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Preferências criadas com sucesso!"})
}

func GetUserPreferences(c *echo.Context, db *database.Database) error{
	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	err := userID.Scan(claims.UserID); if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	existUser, err := db.Query.ExistsUserById(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if !existUser {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Usuário não encontrado."})
	}

	userPreferences, err := db.Query.GetUserPreferences(c.Request().Context(), userID)
	if err != nil{
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, dto.ResponseBase[[]sqlc.GetUserPreferencesRow]{
		Message: "Preferências encontradas com sucesso!",
		Data: userPreferences,
	})
}