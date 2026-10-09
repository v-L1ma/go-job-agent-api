package handlers

import (
	"errors"
	"net/http"
	"time"

	"job-agent-api/internal/database"
	"job-agent-api/internal/dto"
	"job-agent-api/internal/services"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

func CreateSubscription(c *echo.Context, db *database.Database) error {

	return nil
}

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

func UpdateSubscription(c *echo.Context, db *database.Database) error {
	return nil
}

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

	plan, err := db.Query.GetPlanById(c.Request().Context(), subscription.PlanId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if err == nil {
		response.Plan = &dto.PlanSummary{
			Id:          plan.Id.String(),
			Name:        plan.Name,
			Description: plan.Description.String,
			Price:       dto.CentsToBRL(plan.Price),
		}
	}

	return c.JSON(http.StatusOK, dto.ResponseBase[*dto.MySubscriptionResponse]{
		Message: "Assinatura encontrada.",
		Data:    response,
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
