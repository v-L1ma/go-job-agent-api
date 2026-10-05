package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"job-agent-api/internal/database"
	sqlc "job-agent-api/internal/queries"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"
)

func StripeWebhook(c *echo.Context, db *database.Database) error {
	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "failed to read request body",
		})
	}

	signature := c.Request().Header.Get("Stripe-Signature")

	event, err := webhook.ConstructEvent(
		payload,
		signature,
		os.Getenv("STRIPE_WEBHOOK_SECRET"),
	)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid stripe signature",
		})
	}

	switch event.Type {
		case "checkout.session.completed":
			var session stripe.CheckoutSession

			err := json.Unmarshal(event.Data.Raw, &session)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "failed to parse checkout session",
				})
			}

			println("Session ID:", session.ID)
			if session.Customer != nil {
				println("Customer ID:", session.Customer.ID)
			}
			println("checkout.session.completed")

			var sessionId pgtype.UUID
			err = sessionId.Scan(session.ID)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			userId, err := db.Query.GetUserByEmail(c.Request().Context(), pgtype.Text{Valid: true, String: session.CustomerDetails.Email})
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			var planId pgtype.UUID
			err = planId.Scan(session.Metadata["plan_id"])
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			db.Query.CreateSubscription(c.Request().Context(), sqlc.CreateSubscriptionParams{
				Id: sessionId,
				UserId: userId.Id,
				PlanId: planId,
				StripeSubscriptionID: session.Subscription.ID,
				StripeCheckoutSessionID: pgtype.Text{String: session.ID, Valid: true},
				Status: string(session.Status),
			})

			return c.JSON(http.StatusOK, map[string]string{"message": "Subscription created successfully"})

		case "payment_intent.succeeded":
			println("payment_intent.succeeded")

		case "payment_intent.payment_failed":
			println("payment_intent.payment_failed")

		case "customer.subscription.created":
			println("customer.subscription.created")

		case "customer.subscription.updated":
			println("customer.subscription.updated")

		case "customer.subscription.deleted":
			println("customer.subscription.deleted")

		case "price.created", "price.updated":
			println("price.created / price.updated")
			println("Evento recebido:", event.Type)
			var price stripe.Price
			err := json.Unmarshal(event.Data.Raw, &price)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "failed to parse price",
				})
			}

			fmt.Printf("Price: %+v\n", price)

			var productID string
			if price.Product != nil {
				productID = price.Product.ID
			}
			if productID == "" {
				return c.JSON(http.StatusOK, map[string]string{"message": "price without product, skipping"})
			}

			return CreatePlan(c, db, Plan{
				StripeProductID: productID,
				StripePriceID:   price.ID,
				Price:           price.UnitAmount,
				Active:          price.Active,
				DeactivateOnly:  true,
			})
			
		case "product.created", "product.updated":
			println("product.created")
			println("Evento recebido:", event.Type)
			var product stripe.Product
			err := json.Unmarshal(event.Data.Raw, &product)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "failed to parse product",
				})
			}

			fmt.Printf("Product: %+v\n", product)

			var stripePriceID string
			if product.DefaultPrice != nil {
				stripePriceID = product.DefaultPrice.ID
			}

			if !product.Active {
				err := db.Query.DeactivatePlan(c.Request().Context(), sqlc.DeactivatePlanParams{
					StripeProductID: product.ID,
					UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
				})
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
				}
			}

			return CreatePlan(c, db, Plan{
				Name:            product.Name,
				Description:     product.Description,
				StripeProductID: product.ID,
				StripePriceID:   stripePriceID,
				Active:          product.Active,
				RoleId:          DefaultPlanRoleID,
			})

		case "product.deleted":
			println("product.deleted")
			var product stripe.Product
			err := json.Unmarshal(event.Data.Raw, &product)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "failed to parse product",
				})
			}

			return DeactivatePlan(c, db, Plan{
				Name:            product.Name,
				Description:     product.Description,
				StripeProductID: product.ID,
				Active:          product.Active,
				RoleId:          DefaultPlanRoleID,
			})
		default:
			println("Evento não tratado:", string(event.Type))
	}

	return c.NoContent(http.StatusOK)
}