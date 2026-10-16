package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"job-agent-api/internal/database"
	"job-agent-api/internal/dto"
	sqlc "job-agent-api/internal/queries"
	"job-agent-api/internal/services"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/stripe/stripe-go/v86"
)

// GetSubscriptions lista TODAS as assinaturas com dados do usuário e do plano
// (email, nome do plano e nome do cargo via LEFT JOIN), em camelCase, para o
// painel admin.
func GetSubscriptions(c *echo.Context, db *database.Database) error {
	subscriptions, err := db.Query.GetSubscriptions(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	result := make([]dto.AdminSubscriptionResponse, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		result = append(result, dto.MapAdminSubscriptionResponse(subscription))
	}

	// map (e não dto.ResponseBase) para garantir que listas vazias sejam
	// emitidas como "data": [] — ResponseBase usa omitempty e omitiria o campo.
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Subscriptions listed successfully",
		"data":    result,
	})
}

// errResponseSent indica que a resposta HTTP já foi escrita no contexto
// pelo helper — o handler deve apenas retornar nil.
var errResponseSent = errors.New("resposta já enviada")

// loadSubscriptionForUser carrega a assinatura mais recente do usuário
// autenticado.
//
// Ownership: o dono é resolvido SEMPRE pelo userId do JWT (claim "user"),
// nunca por path/query param — assim nenhum usuário consegue ler ou mexer na
// assinatura de outra pessoa.
//
// Retorno:
//   - (sub, nil)           → assinatura encontrada;
//   - (sub, pgx.ErrNoRows) → usuário sem assinatura (o caller decide o
//     status: 404 nas mutações, 200 com lista vazia em /invoices);
//   - (sub, errResponseSent) → claim ausente/inválido; 401 ou 400 já foram
//     respondidos aqui;
//   - (sub, err)           → erro de banco (caller responde 500).
func loadSubscriptionForUser(c *echo.Context, db *database.Database) (sqlc.Subscription, error) {
	claims, ok := c.Get("user").(*services.Claims)
	if !ok {
		_ = c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return sqlc.Subscription{}, errResponseSent
	}

	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		_ = c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return sqlc.Subscription{}, errResponseSent
	}

	subscription, err := db.Query.GetSubscriptionByUserId(c.Request().Context(), userID)
	if err != nil {
		return sqlc.Subscription{}, err
	}
	return subscription, nil
}

// buildMySubscriptionResponse monta o DTO de GET /subscriptions/me a partir
// da linha de GetSubscriptionByUserId (assinatura + resumo do plano, com o
// preço convertido de centavos para reais).
func buildMySubscriptionResponse(ctx context.Context, db *database.Database, subscription sqlc.Subscription) (*dto.MySubscriptionResponse, error) {
	response := &dto.MySubscriptionResponse{
		Id:                subscription.Id.String(),
		PlanId:            subscription.PlanId.String(),
		Status:            subscription.Status,
		CancelAtPeriodEnd: subscription.CancelAtPeriodEnd,
	}

	if subscription.CurrentPeriodEnd.Valid {
		periodEnd := subscription.CurrentPeriodEnd.Time.Format(time.RFC3339)
		response.CurrentPeriodEnd = &periodEnd
	}

	plan, err := db.Query.GetPlanById(ctx, subscription.PlanId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		response.Plan = &dto.PlanSummary{
			Id:          plan.Id.String(),
			Name:        plan.Name,
			Description: plan.Description.String,
			Price:       dto.CentsToBRL(plan.Price),
		}
	}

	return response, nil
}

// GetMySubscription retorna a assinatura mais recente do usuário
// autenticado (mesmo shape usado por cancel/resume).
func GetMySubscription(c *echo.Context, db *database.Database) error {
	claims, ok := c.Get("user").(*services.Claims)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var userId pgtype.UUID
	if err := userId.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user id"})
	}

	subscription, err := db.Query.GetSubscriptionByUserId(c.Request().Context(), userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusOK, dto.ResponseBase[*dto.MySubscriptionResponse]{
				Message: "Nenhuma assinatura encontrada.",
				Data:    nil,
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	response, err := buildMySubscriptionResponse(c.Request().Context(), db, subscription)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, dto.ResponseBase[*dto.MySubscriptionResponse]{
		Message: "Assinatura encontrada.",
		Data:    response,
	})
}

// setCancelAtPeriodEnd agenda (cancel=true) ou reverte (cancel=false) o
// cancelamento para o FIM do período atual, direto na API do Stripe.
//
// O Stripe é a fonte da verdade deste estado: o banco só é sincronizado pelo
// webhook `customer.subscription.updated`, por isso a resposta usa o valor
// recém-devolvido pela API (e não o campo desatualizado da tabela).
//
// Erros: 401/400 (claim ausente/inválido), 404 (usuário sem assinatura),
// 400 (assinatura sem id no Stripe — checkout ainda não concluído),
// 500 (falha na API do Stripe ou no banco).
func setCancelAtPeriodEnd(c *echo.Context, db *database.Database, cancel bool) error {
	subscription, err := loadSubscriptionForUser(c, db)
	if err != nil {
		if errors.Is(err, errResponseSent) {
			return nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "subscription not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if subscription.StripeSubscriptionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "subscription has no stripe id"})
	}

	sc := stripe.NewClient(services.StripeSecretKey())
	updated, err := sc.V1Subscriptions.Update(c.Request().Context(), subscription.StripeSubscriptionID, &stripe.SubscriptionUpdateParams{
		CancelAtPeriodEnd: stripe.Bool(cancel),
	})
	if err != nil {
		// Detalhes do Stripe ficam no log do servidor; o cliente recebe um
		// erro genérico.
		log.Printf("subscriptions: erro ao definir cancelAtPeriodEnd=%v em %s: %v", cancel, subscription.StripeSubscriptionID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update subscription"})
	}

	response, err := buildMySubscriptionResponse(c.Request().Context(), db, subscription)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	response.CancelAtPeriodEnd = updated.CancelAtPeriodEnd

	message := "Cancelamento agendado para o fim do período."
	if !cancel {
		message = "Cancelamento removido."
	}

	return c.JSON(http.StatusOK, dto.ResponseBase[*dto.MySubscriptionResponse]{
		Message: message,
		Data:    response,
	})
}

// CancelMySubscription implementa POST /subscriptions/me/cancel.
// Sem body: marca a assinatura como cancel-at-period-end (reversível).
func CancelMySubscription(c *echo.Context, db *database.Database) error {
	return setCancelAtPeriodEnd(c, db, true)
}

// ResumeMySubscription implementa POST /subscriptions/me/resume.
// Sem body: desfaz o cancelamento agendado (o período segue ativo).
func ResumeMySubscription(c *echo.Context, db *database.Database) error {
	return setCancelAtPeriodEnd(c, db, false)
}

// CreateBillingPortalSession implementa POST /subscriptions/me/portal-session.
//
// Cria uma sessão do Billing Portal do Stripe (gestão de cartão, troca de
// plano e cancelamento na própria UI do Stripe). O ReturnURL aponta para
// {FRONTEND_URL}/perfil — o usuário volta ao app depois do portal.
//
// Erros: 401/400 (claim), 404 (sem assinatura), 400 (sem Stripe Customer),
// 500 (falha na API do Stripe/banco).
func CreateBillingPortalSession(c *echo.Context, db *database.Database) error {
	subscription, err := loadSubscriptionForUser(c, db)
	if err != nil {
		if errors.Is(err, errResponseSent) {
			return nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "subscription not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	ctx := c.Request().Context()

	// Cliente Stripe: primeiro o vinculado à assinatura; fallback para o
	// mapeamento persistido em StripeCustomers (criado no checkout).
	customerID := subscription.StripeCustomerID
	if customerID == "" {
		customer, cErr := db.Query.GetStripeCustomerByUserId(ctx, subscription.UserId)
		if cErr != nil {
			if !errors.Is(cErr, pgx.ErrNoRows) {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": cErr.Error()})
			}
		} else {
			customerID = customer.StripeCustomerId
		}
	}
	if customerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "subscription has no stripe customer"})
	}

	sc := stripe.NewClient(services.StripeSecretKey())
	session, err := sc.V1BillingPortalSessions.Create(ctx, &stripe.BillingPortalSessionCreateParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(services.FrontendURL() + "/perfil"),
	})
	if err != nil {
		log.Printf("subscriptions: erro ao criar billing portal session para %s: %v", customerID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create billing portal session"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Sessão do portal de cobrança criada com sucesso.",
		"data":    map[string]string{"url": session.URL},
	})
}

// GetMyInvoices implementa GET /subscriptions/me/invoices.
//
// Lista as faturas (tabela StripeInvoices) da assinatura do usuário
// autenticado — mais recentes primeiro, no máximo 20.
//
// Contrato: `data` é SEMPRE uma lista (`[]` quando vazia, nunca null e
// nunca ausente) — inclusive quando o usuário não tem assinatura (200 + []).
func GetMyInvoices(c *echo.Context, db *database.Database) error {
	// make com len 0 → sempre emite "data": [], nunca null.
	invoices := make([]dto.InvoiceResponse, 0)

	subscription, err := loadSubscriptionForUser(c, db)
	switch {
	case errors.Is(err, errResponseSent):
		// 401/400 já respondidos pelo helper.
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		// Sem assinatura → 200 com lista vazia.
	case err == nil:
		// subscription.StripeSubscriptionID vazio (checkout ainda não
		// concluído) vira NULL no filtro → nenhuma fatura.
		rows, qErr := db.Query.GetInvoicesByStripeSubscriptionId(c.Request().Context(), pgtype.Text{
			String: subscription.StripeSubscriptionID,
			Valid:  subscription.StripeSubscriptionID != "",
		})
		if qErr != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": qErr.Error()})
		}
		for _, invoice := range rows {
			invoices = append(invoices, dto.MapInvoiceResponse(invoice))
		}
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// map (e não dto.ResponseBase): Data tem `json:"data,omitempty"` e
	// OMITIRIA o campo quando a lista é vazia — o contrato exige "data": [].
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Faturas listadas com sucesso.",
		"data":    invoices,
	})
}

// GetSubscriptionDetails retorna o detalhe de UMA assinatura para o painel
// admin: assinatura (mesmo shape da lista) + plano + usuário + últimas
// faturas (StripeInvoices). Endpoint SOMENTE LEITURA: webhooks do Stripe
// continuam sendo a fonte da verdade de status/período.
//
// Erros: 400 (subscriptionId inválido), 404 (não encontrada), 500 (interno).
func GetSubscriptionDetails(c *echo.Context, db *database.Database) error {
	var subscriptionId pgtype.UUID
	if err := subscriptionId.Scan(c.Param("subscriptionId")); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid subscription id"})
	}

	subscription, err := db.Query.GetSubscriptionById(c.Request().Context(), subscriptionId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "subscription not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Mesmo mapper/shape da lista GET /admin/subscriptions.
	subscriptionResponse := dto.MapAdminSubscriptionResponseFromSubscription(subscription)

	// Plano (defensivo: null se o plano não existir mais).
	var planResponse *dto.PlanResponse
	plan, err := db.Query.GetPlanById(c.Request().Context(), subscription.PlanId)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	} else {
		mappedPlan := dto.MapPlanResponseFromGetPlanById(plan)
		planResponse = &mappedPlan
	}

	// Usuário (defensivo: null — não deve acontecer, há FK para AspNetUsers).
	// Apenas id/name/email/role; PasswordHash nunca é exposto.
	var userResponse *dto.AdminUserResponse
	user, err := db.Query.GetUserById(c.Request().Context(), subscription.UserId)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	} else {
		mappedUser := dto.MapAdminUserResponse(user)
		userResponse = &mappedUser
	}

	// Faturas da assinatura Stripe (últimas 20, ORDER BY CreatedAt DESC).
	// subscription.StripeSubscriptionID vazio (checkout ainda não concluído)
	// vira NULL no filtro → nenhuma fatura.
	invoices, err := db.Query.GetInvoicesByStripeSubscriptionId(c.Request().Context(), pgtype.Text{
		String: subscription.StripeSubscriptionID,
		Valid:  subscription.StripeSubscriptionID != "",
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// make com len 0 → sempre emite "invoices": [], nunca null.
	invoiceResponses := make([]dto.InvoiceResponse, 0, len(invoices))
	for _, invoice := range invoices {
		invoiceResponses = append(invoiceResponses, dto.MapInvoiceResponse(invoice))
	}

	// map (e não dto.ResponseBase) para manter o envelope dos handlers admin.
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Subscription details retrieved successfully",
		"data": dto.SubscriptionDetailResponse{
			Subscription: subscriptionResponse,
			Plan:         planResponse,
			User:         userResponse,
			Invoices:     invoiceResponses,
		},
	})
}

// GetUserSubscriptions retorna o histórico de assinaturas de UM usuário para
// o painel admin (mais recente primeiro), com um resumo do usuário.
// Endpoint SOMENTE LEITURA.
//
// Erros: 400 (userId inválido), 404 (usuário não existe), 500 (interno).
func GetUserSubscriptions(c *echo.Context, db *database.Database) error {
	var userId pgtype.UUID
	if err := userId.Scan(c.Param("userId")); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user id"})
	}

	user, err := db.Query.GetUserById(c.Request().Context(), userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	subscriptions, err := db.Query.GetSubscriptionsByUserId(c.Request().Context(), userId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// make com len 0 → sempre emite "subscriptions": [], nunca null.
	result := make([]dto.AdminSubscriptionResponse, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		result = append(result, dto.MapAdminSubscriptionResponseFromGetSubscriptionsByUserId(subscription))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User subscriptions retrieved successfully",
		"data": dto.UserSubscriptionsResponse{
			User:          dto.MapAdminUserResponse(user),
			Subscriptions: result,
		},
	})
}
