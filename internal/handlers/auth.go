package handlers

import (
	"job-agent-api/internal/database"
	"job-agent-api/internal/database/sqlc"
	"job-agent-api/internal/dto"
	"job-agent-api/internal/helpers"
	"job-agent-api/internal/services"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func RefreshToken(c *echo.Context, db *database.Database) error {
	var req dto.RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.Token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Token é obrigatório."})
	}

	claims := &services.Claims{}
	token, err := jwt.ParseWithClaims(req.Token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Token inválido."})
	}

	newToken, err := services.GenerateTokenFromClaims(claims)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Token renovado com sucesso!",
		"token":   newToken,
	})
}

func ForgotPassword(c *echo.Context, db *database.Database) error {
	var req dto.ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": helpers.ValidationErrors(err),
		})
	}

	email := pgtype.Text{String: req.Email, Valid: true}
	userExists, err := db.Query.ExistsUserByEmail(c.Request().Context(), email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if !userExists {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Se o e-mail informado estiver cadastrado, você receberá um link para redefinir sua senha.",
		})
	}

	rawToken, err := services.GenerateRandomToken(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	tokenHash := services.HashToken(rawToken)

	err = db.Query.CreatePasswordResetToken(c.Request().Context(), sqlc.CreatePasswordResetTokenParams{
		Email:     req.Email,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(1 * time.Hour), Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message":    "Se o e-mail informado estiver cadastrado, você receberá um link para redefinir sua senha.",
		"resetToken": rawToken,
	})
}

func ResetPassword(c *echo.Context, db *database.Database) error {
	var req dto.ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": helpers.ValidationErrors(err),
		})
	}

	if req.NewPassword != req.ConfirmPassword {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "As senhas não coincidem."})
	}

	tokenHash := services.HashToken(req.Token)

	resetToken, err := db.Query.GetPasswordResetTokenByHash(c.Request().Context(), tokenHash)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Token inválido ou expirado."})
	}

	passwordHash, err := services.HashPassword(req.NewPassword)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	userEmail := pgtype.Text{String: resetToken.Email, Valid: true}
	err = db.Query.UpdateUserPasswordByEmail(c.Request().Context(), sqlc.UpdateUserPasswordByEmailParams{
		Email:        userEmail,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	err = db.Query.MarkResetTokenAsUsed(c.Request().Context(), resetToken.Id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Senha redefinida com sucesso!"})
}