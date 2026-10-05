package handlers

import (
	"fmt"
	"job-agent-api/internal/database"
	"job-agent-api/internal/services"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/stripe/stripe-go/v86"
)

func CreatePaymentSession(c *echo.Context, db *database.Database) error {
	// Implementation for creating a payment session
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

	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	err = userID.Scan(claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	sc := stripe.NewClient(os.Getenv("STRIPE_API_KEY"))
	params := &stripe.CheckoutSessionCreateParams{
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			&stripe.CheckoutSessionCreateLineItemParams{
			Price: stripe.String(plan.StripePriceID.String),
			Quantity: stripe.Int64(1),
			},
		},
		Mode: stripe.String(stripe.CheckoutSessionModePayment),
		UIMode: stripe.String(stripe.CheckoutSessionUIModeForm),
		ReturnURL: stripe.String("https://trampo.click"),
		IntegrationIdentifier: stripe.String("embedded_web_0001"),
		Metadata: map[string]string{
			"user_id": userID.String(),
			"plan_id": plan.Id.String(),
		},
	}
	result, err := sc.V1CheckoutSessions.Create(c.Request().Context(), params)
	if err != nil {
		return err
	}
	// Do something with the created payment session, e.g., store it in the database
	_ = result

	fmt.Println("Payment session created:", result.ClientSecret)

	return c.JSON(200, map[string]string{
		"message": "Sessão de pagamento criada com sucesso",
		"data": result.ClientSecret,
	})
}