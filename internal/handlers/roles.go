package handlers

import (
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

func getCurrentUserId(c *echo.Context) (pgtype.UUID, error) {
	claims := c.Get("user").(*services.Claims)
	var userId pgtype.UUID
	if err := userId.Scan(claims.UserID); err != nil {
		return userId, err
	}
	return userId, nil
}

func parseRoleId(c *echo.Context) (pgtype.UUID, error) {
	var roleId pgtype.UUID
	if err := roleId.Scan(c.Param("id")); err != nil {
		return roleId, err
	}
	return roleId, nil
}

func mapRoleResponse(role sqlc.Role) dto.RoleResponse {
	return dto.RoleResponse{
		Id:             role.Id.String(),
		Name:           role.Name,
		Active:         role.Active,
		CreatedBy:      role.CreatedBy,
		CreatedAt:      role.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		LastModifiedBy: role.LastModifiedBy,
		LastModifiedAt: role.LastModifiedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func GetRoles(c *echo.Context, db *database.Database) error {
	roles, err := db.Query.GetRoles(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	result := make([]dto.RoleResponse, 0, len(roles))
	for _, role := range roles {
		result = append(result, mapRoleResponse(role))
	}

	return c.JSON(http.StatusOK, dto.ResponseBase[[]dto.RoleResponse]{
		Message: "Cargos listados com sucesso!",
		Data:    result,
	})
}

func GetRoleById(c *echo.Context, db *database.Database) error {
	roleId, err := parseRoleId(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ID de cargo inválido"})
	}

	role, err := db.Query.GetRoleById(c.Request().Context(), roleId)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Cargo não encontrado"})
	}

	return c.JSON(http.StatusOK, dto.ResponseBase[dto.RoleResponse]{
		Message: "Cargo encontrado com sucesso!",
		Data:    mapRoleResponse(role),
	})
}

func CreateRole(c *echo.Context, db *database.Database) error {
	userId, err := getCurrentUserId(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req dto.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": helpers.ValidationErrors(err),
		})
	}

	exists, err := db.Query.ExistsRoleByName(c.Request().Context(), sqlc.ExistsRoleByNameParams{
		Name: req.Name,
		ID: pgtype.UUID{Valid: false},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if exists {
		return c.JSON(http.StatusConflict, map[string]string{"error": "Cargo já existe"})
	}

	if err := db.Query.CreateRole(c.Request().Context(), sqlc.CreateRoleParams{
		Name: req.Name,
		Active: true,
		CreatedBy: userId.String(),
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		LastModifiedBy: userId.String(),
		LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Cargo criado com sucesso!",
	})
}

func UpdateRole(c *echo.Context, db *database.Database) error {
	userId, err := getCurrentUserId(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req dto.UpdateRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": helpers.ValidationErrors(err),
		})
	}

	roleId, err := parseRoleId(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ID de cargo inválido"})
	}

	exists, err := db.Query.ExistsRoleById(c.Request().Context(), roleId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Cargo não encontrado"})
	}

	nameInUse, err := db.Query.ExistsRoleByName(c.Request().Context(), sqlc.ExistsRoleByNameParams{
		Name: req.Name,
		ID:   roleId,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if nameInUse {
		return c.JSON(http.StatusConflict, map[string]string{"error": "Já existe um cargo com este nome"})
	}

	if err := db.Query.UpdateRole(c.Request().Context(), sqlc.UpdateRoleParams{
		Name: req.Name,
		Active: req.Active,
		LastModifiedBy: userId.String(),
		LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Id: roleId,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Cargo atualizado com sucesso!",
	})
}

func DeleteRole(c *echo.Context, db *database.Database) error {
	userId, err := getCurrentUserId(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	roleId, err := parseRoleId(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ID de cargo inválido"})
	}

	exists, err := db.Query.ExistsRoleById(c.Request().Context(), roleId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Cargo não encontrado"})
	}

	if err := db.Query.DeactivateRole(c.Request().Context(), sqlc.DeactivateRoleParams{
		Id:             roleId,
		LastModifiedBy: userId.String(),
		LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Cargo desativado com sucesso!",
	})
}