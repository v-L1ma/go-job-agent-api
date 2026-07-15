package handlers

import (
	"slices"
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

type SetUserPreferencesRequest struct {
	Skills []string `json:"skills" validate:"min=1,dive,required"`
	Levels []string `json:"levels" validate:"min=1,dive,required"`
}

func SetUserPreferences(c *echo.Context, db *database.Database) error {
	claims := c.Get("user").(*services.Claims)

	var cvId pgtype.UUID
	if err := cvId.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req SetUserPreferencesRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": helpers.ValidationErrors(err),
		})
	}

	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	existUser, err := db.Query.ExistsUserById(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if !existUser {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Usuário não encontrado."})
	}

	if len(req.Skills) <= 0 && len(req.Levels) <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Informe ao menos uma Habilidade e uma senioridade."})
	}

	alreadyCreated, err := db.Query.ExistsSearchQueryByUserId(c.Request().Context(), userID)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	slices.Sort(req.Skills)
	
	slices.Sort(req.Levels)

	var query strings.Builder
	query.WriteString("(")

	for i, skill := range req.Skills {
		if i > 0{
			query.WriteString(" OR ")
		}
		query.WriteString(skill)
	}

	query.WriteString(") AND (")

	for i, level := range req.Levels {
		if i > 0{
			query.WriteString(" OR ")
		}
		query.WriteString(level)
	}

	query.WriteString(")")

	var normalizedHash strings.Builder
	
	for i, skill := range req.Skills {
		if i > 0 {
			normalizedHash.WriteString("-")
		}
		normalizedHash.WriteString(skill)
	}
	for i, level := range req.Levels {
		if i > 0 || len(req.Skills) > 0 {
			normalizedHash.WriteString("-")
		}
		normalizedHash.WriteString(level)
	}

	if alreadyCreated {
		currentSearchQueryId, err := db.Query.GetSearchQueryIdByUserId(c.Request().Context(), userID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		usedByAnotherUser, err := db.Query.ExistsSearchQueryByAnotherUser(c.Request().Context(), sqlc.ExistsSearchQueryByAnotherUserParams{
			SearchQueryId: currentSearchQueryId,
			UserId: userID,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		if !usedByAnotherUser {
			err := db.Query.UpdateSearchQuery(c.Request().Context(), sqlc.UpdateSearchQueryParams{
				Query: query.String(),
				Levels: req.Levels,
				Keywords: req.Skills,
				NormalizedHash: normalizedHash.String(),
				LastModifiedBy: userID.String(),
				LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
				UserId: userID,
			})
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error update": err.Error()})
			}

			return c.JSON(http.StatusOK, map[string]string{"message": "Preferências atualizadas com sucesso!"})
		}

		searchQueryId, err := db.Query.CreateSearchQuery(c.Request().Context(), sqlc.CreateSearchQueryParams{
			Query:          query.String(),
			Keywords:       req.Skills,
			NormalizedHash: normalizedHash.String(),
			Area:             "",
			Levels:           req.Levels,
			Active:           true,
			CreatedBy:      userID.String(),
			CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
			LastModifiedBy: userID.String(),
			LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error create": err.Error()})
		}

		err = db.Query.UpdateUserSearchQuery(c.Request().Context(), sqlc.UpdateUserSearchQueryParams{
			UserId: userID,
			SearchQueryId: searchQueryId,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error update user search query": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Preferências atualizadas com sucesso!"})
	}

	searchQueryId, err := db.Query.CreateSearchQuery(c.Request().Context(), sqlc.CreateSearchQueryParams{
		Query:          query.String(),
		Keywords:       req.Skills,
		NormalizedHash: normalizedHash.String(),
		Area:             "",
		Levels:           req.Levels,
		Active:           true,
		CreatedBy:      userID.String(),
		CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		LastModifiedBy: userID.String(),
		LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error create": err.Error()})
	}

	err = db.Query.CreateUserSearchQuery(c.Request().Context(), sqlc.CreateUserSearchQueryParams{
		UserId: userID,
		SearchQueryId: searchQueryId,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error create user search query": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Preferências criadas com sucesso!"})
}

type UpdateProfileRequest struct {
	Name  string `json:"name" validate:"min=6,max=50"`
	Email string `json:"email" validate:"email,min=6,max=50"`
}

type ChangePasswordRequest struct {
	CurrentPassword    string `json:"currentPassword"`
	NewPassword        string `json:"newPassword"`
	ConfirmNewPassword string `json:"confirmNewPassword"`
}

func GetUserProfile(c *echo.Context, db *database.Database) error {
	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	user, err := db.Query.GetUserById(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Usuário não encontrado."})
	}

	return c.JSON(http.StatusOK, dto.ResponseBase[map[string]any]{
		Message: "Perfil encontrado com sucesso!",
		Data: map[string]any{
			"id":    user.Id.String(),
			"name":  user.Name,
			"email": user.Email.String,
			"cpf":   user.CPF,
		},
	})
}

func UpdateProfile(c *echo.Context, db *database.Database) error {
	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	existUser, err := db.Query.ExistsUserById(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !existUser {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Usuário não encontrado."})
	}

	var email pgtype.Text
	if req.Email != "" {
		email.Scan(req.Email)
	}

	user, err := db.Query.ExistsUserByEmail(c.Request().Context(), pgtype.Text{String: req.Email, Valid: true})

	if user && !strings.EqualFold(claims.Email, req.Email) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Este e-mail já está em uso."})
	} //todo terminar de validar se um email ja esta sendo utilizado

	err = db.Query.UpdateUser(c.Request().Context(), sqlc.UpdateUserParams{
		Id:    userID,
		Name:  req.Name,
		CPF:   "",
		Email: email,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Perfil atualizado com sucesso!"})
}

func ChangePassword(c *echo.Context, db *database.Database) error {
	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.NewPassword != req.ConfirmNewPassword {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "As senhas não coincidem."})
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Preencha todos os campos."})
	}

	user, err := db.Query.GetUserById(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Usuário não encontrado."})
	}

	err = services.CheckPassword(req.CurrentPassword, user.PasswordHash.String)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Senha atual incorreta."})
	}

	passwordHash, err := services.HashPassword(req.NewPassword)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	err = db.Query.UpdateUserPassword(c.Request().Context(), sqlc.UpdateUserPasswordParams{
		Id:           userID,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Senha alterada com sucesso!"})
}

func GetUserStatistics(c *echo.Context, db *database.Database) error {
	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	ctx := c.Request().Context()

	stats, err := db.Query.GetUserJobStats(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	total := int(stats.Total)
	applied := int(stats.Applied)
	skipped := int(stats.Skipped)

	prevMonthTotal := int(0)
	prevMonthCount, err := db.Query.GetPrevMonthJobCount(ctx, userID)
	if err == nil {
		prevMonthTotal = int(prevMonthCount)
	}

	variation := 0
	if prevMonthTotal > 0 {
		variation = ((total - prevMonthTotal) * 100) / prevMonthTotal
	}

	successRate := 0
	if total > 0 {
		successRate = (applied * 100) / total
	}

	skippedPercentage := 0
	if total > 0 {
		skippedPercentage = (skipped * 100) / total
	}

	type PerDay struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}

	perDayRows, err := db.Query.GetApplicationsPerDay(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var applicationsPerDay []PerDay
	for _, row := range perDayRows {
		applicationsPerDay = append(applicationsPerDay, PerDay{
			Date:  row.Date.Time.Format("02/01"),
			Count: int(row.Count),
		})
	}

	type PlatformCount struct {
		Platform string `json:"platform"`
		Count    int    `json:"count"`
	}

	platformRows, err := db.Query.GetPlatformDistribution(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var platformDistribution []PlatformCount
	for _, row := range platformRows {
		platformDistribution = append(platformDistribution, PlatformCount{
			Platform: row.Platform,
			Count:    int(row.Count),
		})
	}

	data := map[string]any{
		"total": map[string]any{
			"count":          total,
			"variation":      variation,
			"variationLabel": "vs mês passado",
		},
		"applied": map[string]any{
			"count":       applied,
			"successRate": successRate,
		},
		"skipped": map[string]any{
			"count": skipped,
			"label": "Filtros aplicados",
		},
		"failures": map[string]any{
			"count":    0,
			"thisWeek": 0,
		},
		"applicationsPerDay":   applicationsPerDay,
		"platformDistribution": platformDistribution,
		"statusDistribution": []map[string]any{
			{"status": "Total", "count": total},
			{"status": "Aplicadas", "count": applied, "percentage": successRate},
			{"status": "Puladas", "count": skipped, "percentage": skippedPercentage},
			{"status": "Falhas", "count": 0, "percentage": 0},
		},
		"recentApplications": []any{},
	}

	return c.JSON(http.StatusOK, dto.ResponseBase[map[string]any]{
		Message: "Estatísticas encontradas com sucesso!",
		Data:    data,
	})
}

func GetUserPreferences(c *echo.Context, db *database.Database) error {
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

	userPreferences, err := db.Query.GetUserPreferences(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, dto.ResponseBase[[]sqlc.GetUserPreferencesRow]{
		Message: "Preferências encontradas com sucesso!",
		Data:    userPreferences,
	})
}
