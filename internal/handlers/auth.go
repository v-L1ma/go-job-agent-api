package handlers

import (
	"job-agent-api/internal/database"
	"job-agent-api/internal/database/sqlc"
	"job-agent-api/internal/dto"
	"job-agent-api/internal/helpers"
	"job-agent-api/internal/services"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

func Login(c *echo.Context, db *database.Database) error{
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	var email pgtype.Text
	if err := email.Scan(req.Email); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	user, err := db.Query.GetUserByEmail(c.Request().Context(), email)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err = services.CheckPassword(
		req.Password,
		user.PasswordHash.String,
	)

	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email e/ou senha inválidos."})
	}

	token, err := services.GenerateToken(user)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Login efetuado com sucesso!",
		"token" : token,
	})
}

func Register(c *echo.Context, db *database.Database) error{
	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil{
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(&req); err != nil{
		return c.JSON(http.StatusBadRequest,map[string]any{
			"errors": helpers.ValidationErrors(err),
		})
	}

	if req.Password != req.ConfirmPassword{
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "As senhas não coincidem."})
	}

	user, err := db.Query.ExistsUserByEmail(c.Request().Context(), pgtype.Text{String: req.Email, Valid: true})

	if user {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Este e-mail já está em uso."})
	}

	passwordHash, err := services.HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err = db.Query.CreateUser(c.Request().Context(), sqlc.CreateUserParams{
		Name: req.Name,
		CPF: "",
		Email: pgtype.Text{String: req.Email, Valid: true},
		TwoFactorEnabled: false,
		EmailConfirmed: true,
		LockoutEnabled: false,
		AccessFailedCount: 0,
		OnboardingCompleted: false,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"Message": "Usuário criado com sucesso."})
}