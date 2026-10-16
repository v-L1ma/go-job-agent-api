package handlers

import (
	"errors"
	"fmt"
	"html"
	"job-agent-api/internal/database"
	"job-agent-api/internal/dto"
	"job-agent-api/internal/helpers"
	sqlc "job-agent-api/internal/queries"
	"job-agent-api/internal/services"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/auth/credentials/idtoken"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

func Login(c *echo.Context, db *database.Database) error {
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email e/ou senha inválidos."})
	}

	err = services.CheckPassword(
		req.Password,
		user.PasswordHash.String,
	)

	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email e/ou senha inválidos."})
	}

	// Bloqueia o login enquanto a conta não tiver o e-mail confirmado.
	// O token JWT só é emitido depois da confirmação.
	if !user.EmailConfirmed {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "E-mail não confirmado. Verifique sua caixa de entrada ou reenvie o link de ativação."})
	}

	hasUploadedCv, err := db.Query.ExistsUploadedCv(c.Request().Context(), user.Id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	preferences, err := db.Query.GetUserPreferences(c.Request().Context(), user.Id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	hasCompletedOnboarding := hasUploadedCv && len(preferences.Keywords) > 0 && len(preferences.Levels) > 0

	token, err := services.GenerateToken(user, hasCompletedOnboarding)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message":             "Login efetuado com sucesso!",
		"token":               token,
	})
}

func LoginGoogle(c *echo.Context, db *database.Database) error {
	var req dto.LoginGoogleRequest

	if err := c.Bind(&req); err != nil {
		fmt.Printf("Erro ao fazer bind do request: %v\n", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "requisição inválida",
		})
	}

	if req.IDToken == "" {
		fmt.Print("Id nao fornecido")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "idToken é obrigatório",
		})
	}
	fmt.Printf("ID Token: %s\n", req.IDToken)

	payload, err := idtoken.Validate(
		c.Request().Context(),
		req.IDToken,
		os.Getenv("GOOGLE_CLIENT_ID"),
	)

	if err != nil {
		fmt.Printf("Erro ao validar o ID Token: %v\n", err)
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Google ID Token inválido",
		})
	}

	// Dados validados pelo Google
	googleID := payload.Subject
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	// Aqui você procura/cria o usuário no seu banco.
	// ...
	fmt.Printf("")
	fmt.Printf("Google ID: %s, Email: %s, Name: %s, Picture: %s\n", googleID, email, name, picture)
	fmt.Printf("")

	user, err := db.Query.GetUserByEmail(c.Request().Context(), pgtype.Text{String: email, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {

			_, err := db.Query.CreateUser(c.Request().Context(), sqlc.CreateUserParams{
				Name:                name,
				CPF:                 "",
				Email:               pgtype.Text{String: email, Valid: true},
				TwoFactorEnabled:    false,
				EmailConfirmed:      true,
				LockoutEnabled:      false,
				AccessFailedCount:   0,
				OnboardingCompleted: false,
				PasswordHash:        pgtype.Text{String: "", Valid: true},
			})
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			role, err := db.Query.GetRoleByName(c.Request().Context(), "Standard")
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			userByEmail, err := db.Query.GetUserByEmail(c.Request().Context(), pgtype.Text{String: email, Valid: true})
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			err = db.Query.AssignRoleToUser(c.Request().Context(), sqlc.AssignRoleToUserParams{
				UserId: userByEmail.Id,
				RoleId: role.Id,
			})
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			user = userByEmail
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	hasUploadedCv, err := db.Query.ExistsUploadedCv(c.Request().Context(), user.Id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	preferences, err := db.Query.GetUserPreferences(c.Request().Context(), user.Id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	hasCompletedOnboarding := hasUploadedCv && len(preferences.Keywords) > 0 && len(preferences.Levels) > 0

	token, err := services.GenerateToken(user, hasCompletedOnboarding)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message":             "Login efetuado com sucesso!",
		"token":               token,
	})
}

func Register(c *echo.Context, db *database.Database) error {
	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": helpers.ValidationErrors(err),
		})
	}

	if req.Password != req.ConfirmPassword {
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

	// Token de confirmação de e-mail: 32 bytes aleatórios, guardados APENAS
	// como hash SHA-256 (single-use, expira em 24h). O token cru só sai daqui
	// dentro do e-mail de ativação — nunca na resposta nem em log.
	rawToken, err := services.GenerateRandomToken(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	tokenHash := services.HashToken(rawToken)

	// Usuário + role + token de confirmação em UMA transação: ou tudo é
	// persistido, ou nada (evita conta criada sem token de ativação).
	ctx := c.Request().Context()
	tx, err := db.Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer tx.Rollback(ctx)

	qtx := db.Query.WithTx(tx)

	userId, err := qtx.CreateUser(ctx, sqlc.CreateUserParams{
		Name:                req.Name,
		CPF:                 "",
		Email:               pgtype.Text{String: req.Email, Valid: true},
		TwoFactorEnabled:    false,
		EmailConfirmed:      false,
		LockoutEnabled:      false,
		AccessFailedCount:   0,
		OnboardingCompleted: false,
		PasswordHash:        pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	role, err := qtx.GetRoleByName(ctx, "Standard")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	err = qtx.AssignRoleToUser(ctx, sqlc.AssignRoleToUserParams{
		UserId: userId,
		RoleId: role.Id,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	err = qtx.CreateEmailConfirmationToken(ctx, sqlc.CreateEmailConfirmationTokenParams{
		Email:     req.Email,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Link enviado POR E-MAIL (o frontend consome /verify-email?token=).
	verifyURL := services.FrontendURL() + "/verify-email?token=" + url.QueryEscape(rawToken)

	// Falha de envio NÃO falha o registro: apenas loga e a resposta
	// continua 201. O usuário pode usar /resend-confirmation depois.
	if sendErr := services.NewEmailer().SendWithTemplate(ctx, req.Email, "account-confirmation", map[string]interface{}{
		"LINK": verifyURL,
		"NOME_DO_USUARIO":  req.Name,
	}); sendErr != nil {
		log.Printf("register: falha ao enviar e-mail de confirmação para %q: %v", req.Email, sendErr)
	}

	return c.JSON(http.StatusCreated, map[string]string{"Message": "Usuário criado com sucesso. Verifique seu e-mail para ativar sua conta."})
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

	// Mensagem ÚNICA e idêntica nos dois caminhos (e-mail existe ou não):
	// a resposta nunca revela se a conta existe.
	const forgotPasswordMessage = "Se o e-mail existir, você receberá um link de recuperação."

	email := pgtype.Text{String: req.Email, Valid: true}

	user, err := db.Query.GetUserByEmail(c.Request().Context(), email)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]string{"message": forgotPasswordMessage})
	}

	// Token cru: guardado apenas como hash SHA-256 no banco (single-use,
	// expira em 1h). Ele NUNCA é devolvido na resposta nem escrito em log
	// de request/response — só sai daqui dentro do e-mail.
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

	// Link enviado POR E-MAIL (o frontend consome /reset-password?token=).
	resetURL := services.FrontendURL() + "/reset-password?token=" + url.QueryEscape(rawToken)

	// Lê o template HTML do arquivo e substitui o placeholder pelo link.
	// templatePath := "internal/templates/password_reset.html"
	// templateBytes, err := os.ReadFile(templatePath)
	// if err != nil {
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read email template"})
	// }
	// htmlBody := strings.ReplaceAll(string(templateBytes), "{{.ResetURL}}", resetURL)

	// Falha de envio é apenas logada: a resposta continua 200 com a MESMA
	// mensagem para não vazar a existência da conta.
	if sendErr := services.NewEmailer().SendWithTemplate(
		c.Request().Context(),
		req.Email,
		"password-reset",
		map[string]interface{}{
			"LINK_DE_REDEFINICAO": resetURL,
			"NOME_DO_USUARIO":     user.Name,
		},
	); sendErr != nil {
		log.Printf("forgot-password: falha ao enviar e-mail para %q: %v", req.Email, sendErr)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": forgotPasswordMessage})
}

// VerifyEmail confirma a conta a partir do token enviado por e-mail
// ({FRONTEND_URL}/verify-email?token=...). Rota pública.
//
// A busca do token NÃO filtra por Used/ExpiresAt: a decisão é do handler,
// porque o endpoint é idempotente — scanners de e-mail abrem o link antes
// do usuário e uma conta já confirmada deve responder sucesso mesmo com o
// token gasto ou expirado.
func VerifyEmail(c *echo.Context, db *database.Database) error {
	var req dto.VerifyEmailRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": helpers.ValidationErrors(err),
		})
	}

	ctx := c.Request().Context()
	const confirmedMessage = "E-mail confirmado com sucesso!"

	confirmToken, err := db.Query.GetEmailConfirmationTokenByHash(ctx, services.HashToken(req.Token))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Token inválido ou expirado."})
	}

	userEmail := pgtype.Text{String: confirmToken.Email, Valid: true}

	user, err := db.Query.GetUserByEmail(ctx, userEmail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Token inválido ou expirado."})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Idempotência: conta já confirmada → sucesso, sem consumir o token e
	// independentemente do estado dele (expirado/usado).
	if user.EmailConfirmed {
		return c.JSON(http.StatusOK, map[string]string{"message": confirmedMessage})
	}

	if confirmToken.Used || !confirmToken.ExpiresAt.Time.After(time.Now()) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Token inválido ou expirado."})
	}

	// Transacional: confirmar a conta e consumir o token (single-use)
	// acontecem juntos — ou as duas coisas acontecem, ou nenhuma.
	tx, err := db.Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer tx.Rollback(ctx)

	qtx := db.Query.WithTx(tx)

	if err := qtx.ConfirmUserEmailByEmail(ctx, userEmail); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := qtx.MarkEmailConfirmationTokenAsUsed(ctx, confirmToken.Id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": confirmedMessage})
}

// ResendConfirmation reenvia o link de ativação da conta. Rota pública.
//
// A resposta é SEMPRE a mesma mensagem, exista ou não o e-mail (e esteja ou
// não confirmado) — nunca revela a existência da conta.
func ResendConfirmation(c *echo.Context, db *database.Database) error {
	var req dto.ResendConfirmationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": helpers.ValidationErrors(err),
		})
	}

	ctx := c.Request().Context()
	const resendMessage = "Se o e-mail existir e ainda não estiver confirmado, você receberá um novo link de ativação."

	user, err := db.Query.GetUserByEmail(ctx, pgtype.Text{String: req.Email, Valid: true})
	if err != nil || user.EmailConfirmed {
		// E-mail inexistente ou já confirmado: nada a enviar, resposta 200
		// idêntica ao caso de sucesso.
		return c.JSON(http.StatusOK, map[string]string{"message": resendMessage})
	}

	// Novo token a cada chamada (os anteriores permanecem válidos até
	// expirarem; como a conta ainda não foi confirmada, isso não gera conflito).
	rawToken, err := services.GenerateRandomToken(32)
	if err != nil {
		log.Printf("resend-confirmation: falha ao gerar token para %q: %v", req.Email, err)
		return c.JSON(http.StatusOK, map[string]string{"message": resendMessage})
	}

	err = db.Query.CreateEmailConfirmationToken(ctx, sqlc.CreateEmailConfirmationTokenParams{
		Email:     req.Email,
		TokenHash: services.HashToken(rawToken),
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true},
	})
	if err != nil {
		// Erro interno também não é exposto: devolver 500 aqui só acontece
		// quando o usuário existe e não está confirmado (vazaria a existência).
		log.Printf("resend-confirmation: falha ao criar token para %q: %v", req.Email, err)
		return c.JSON(http.StatusOK, map[string]string{"message": resendMessage})
	}

	verifyURL := services.FrontendURL() + "/verify-email?token=" + url.QueryEscape(rawToken)

	if sendErr := services.NewEmailer().SendWithTemplate(ctx, req.Email, "account-confirmation", map[string]interface{}{
		"LINK": verifyURL,
		"NOME_DO_USUARIO":  user.Name,
	}); sendErr != nil {
		log.Printf("resend-confirmation: falha ao enviar e-mail para %q: %v", req.Email, sendErr)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": resendMessage})
}


// inline CSS, em português. `resetURL` é o link completo com o token.
func passwordResetEmailHTML(resetURL string) string {
	href := html.EscapeString(resetURL)
	return strings.ReplaceAll(`<!DOCTYPE html>
<html lang="pt-BR">
<body style="margin:0;padding:0;background:#f4f5f7;font-family:Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4f5f7;padding:24px 0;">
    <tr>
      <td align="center">
        <table role="presentation" width="560" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;padding:32px;">
          <tr>
            <td style="font-size:20px;font-weight:bold;color:#111827;padding-bottom:16px;">
              Redefini&ccedil;&atilde;o de senha
            </td>
          </tr>
          <tr>
            <td style="font-size:14px;line-height:21px;color:#374151;padding-bottom:24px;">
              Ol&aacute;! Recebemos uma solicita&ccedil;&atilde;o para redefinir a senha da sua conta.
              Para escolher uma nova senha, clique no bot&atilde;o abaixo.
            </td>
          </tr>
          <tr>
            <td style="padding-bottom:24px;">
              <a href="{{URL}}"
                 style="display:inline-block;background:#111827;color:#ffffff;text-decoration:none;font-size:14px;font-weight:bold;padding:12px 24px;border-radius:6px;">
                Redefinir minha senha
              </a>
            </td>
          </tr>
          <tr>
            <td style="font-size:12px;line-height:18px;color:#6b7280;padding-bottom:8px;">
              Este link expira em 1 hora e pode ser usado uma &uacute;nica vez.
              Se o bot&atilde;o n&atilde;o funcionar, copie e cole o endereço abaixo no navegador:
            </td>
          </tr>
          <tr>
            <td style="font-size:12px;line-height:18px;color:#2563eb;word-break:break-all;">
              {{URL}}
            </td>
          </tr>
          <tr>
            <td style="font-size:12px;line-height:18px;color:#6b7280;padding-top:24px;">
              Se voc&ecirc; n&atilde;o solicitou esta altera&ccedil;&atilde;o, ignore este e-mail. Sua senha permanecer&aacute; a mesma.
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, "{{URL}}", href)
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

	// Transacional: trocar a senha e consumir o token (single-use) acontece
	// em UMA transação — ou as duas coisas acontecem, ou nenhuma. Assim um
	// token nunca fica válido depois que a senha já mudou (e vice-versa).
	ctx := c.Request().Context()
	tx, err := db.Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer tx.Rollback(ctx)

	qtx := db.Query.WithTx(tx)

	userEmail := pgtype.Text{String: resetToken.Email, Valid: true}
	if err := qtx.UpdateUserPasswordByEmail(ctx, sqlc.UpdateUserPasswordByEmailParams{
		Email:        userEmail,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := qtx.MarkResetTokenAsUsed(ctx, resetToken.Id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Senha redefinida com sucesso!"})
}
