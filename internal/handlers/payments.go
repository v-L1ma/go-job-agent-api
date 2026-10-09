package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"job-agent-api/internal/database"
	sqlc "job-agent-api/internal/queries"
	"job-agent-api/internal/services"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/stripe/stripe-go/v86"
)

// CreatePaymentSession cria uma Checkout Session (mode=subscription) para o
// plano indicado.
//
// Comportamento:
//   - cria/recupera o Stripe Customer do usuário (tabela StripeCustomers) e
//     reutiliza sempre o mesmo customer;
//   - envia user_id/plan_id na metadata da Session e da Subscription, além de
//     client_reference_id — o webhook identifica o usuário por esses campos
//     (NUNCA por e-mail);
//   - resposta inalterada: {message, data} com data = client secret.
func CreatePaymentSession(c *echo.Context, db *database.Database) error {
	planId := c.Param("planId")
	if planId == "" {
		return c.JSON(400, map[string]string{
			"error": "planId parameter is required",
		})
	}

	var planUUID pgtype.UUID
	err := planUUID.Scan(planId)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "planId parameter is required",
		})
	}

	plan, err := db.Query.GetPlanById(c.Request().Context(), planUUID)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "planId parameter is required",
		})
	}

	if !plan.StripePriceID.Valid || plan.StripePriceID.String == "" {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{
			"error": "plan does not have a stripe price configured",
		})
	}

	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	err = userID.Scan(claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	ctx := c.Request().Context()

	customerID, err := ensureStripeCustomer(ctx, db, userID, claims.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	sc := stripe.NewClient(services.StripeSecretKey())
	params := &stripe.CheckoutSessionCreateParams{
		Customer:          stripe.String(customerID),
		ClientReferenceID: stripe.String(userID.String()),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Price:    stripe.String(plan.StripePriceID.String),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:                  stripe.String(stripe.CheckoutSessionModeSubscription),
		UIMode:                stripe.String(stripe.CheckoutSessionUIModeForm),
		ReturnURL:             stripe.String("http://localhost:3000/perfil"),
		IntegrationIdentifier: stripe.String("embedded_web_0001"),
		Metadata: map[string]string{
			"user_id": userID.String(),
			"plan_id": plan.Id.String(),
		},
		// A metadata da Subscription garante que os eventos
		// customer.subscription.* carreguem o usuário sem depender do e-mail.
		SubscriptionData: &stripe.CheckoutSessionCreateSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": userID.String(),
				"plan_id": plan.Id.String(),
			},
		},
	}

	result, err := sc.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		fmt.Println("Failed to create payment session:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create checkout session"})
	}

	return c.JSON(200, map[string]string{
		"message": "Sessão de pagamento criada com sucesso",
		"data":    result.ClientSecret,
	})
}

// ensureStripeCustomer recupera o Stripe Customer persistido para o usuário
// ou cria um novo (com metadata[user_id]) e persiste o mapeamento.
func ensureStripeCustomer(ctx context.Context, db *database.Database, userID pgtype.UUID, email string) (string, error) {
	existing, err := db.Query.GetStripeCustomerByUserId(ctx, userID)
	if err == nil {
		return existing.StripeCustomerId, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	sc := stripe.NewClient(services.StripeSecretKey())
	params := &stripe.CustomerCreateParams{
		Metadata: map[string]string{
			"user_id": userID.String(),
		},
	}
	if email != "" {
		params.Email = stripe.String(email)
	}

	customer, err := sc.V1Customers.Create(ctx, params)
	if err != nil {
		return "", err
	}

	if err := db.Query.UpsertStripeCustomer(ctx, sqlc.UpsertStripeCustomerParams{
		UserId:           userID,
		StripeCustomerId: customer.ID,
	}); err != nil {
		return "", err
	}

	return customer.ID, nil
}
