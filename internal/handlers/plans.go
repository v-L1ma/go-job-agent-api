package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"job-agent-api/internal/database"
	"job-agent-api/internal/dto"
	sqlc "job-agent-api/internal/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

const DefaultPlanRoleID = "a2238759-645f-4d7e-867e-1ccf3d86afef"

type Plan struct {
	Name            string
	Description     string
	StripePriceID   string
	StripeProductID string
	Price           int64
	Active          bool
	RoleId          string
	// DeactivateOnly indica que o evento so pode desativar o plano,
	// nunca reativar (ex.: eventos de price nao devem reativar um plano
	// desativado pelo produto).
	DeactivateOnly bool
}

// CreatePlan cria ou atualiza um plano a partir de um evento do Stripe.
// Recebe um stripeWebhookStore para que rode dentro da transação do webhook.
func CreatePlan(ctx context.Context, store stripeWebhookStore, plan Plan) (string, error) {
	alreadyExists, err := store.ExistsPlanByProductId(ctx, plan.StripeProductID)
	if err != nil {
		return "", err
	}

	if alreadyExists {
		return MergePlan(ctx, store, plan)
	}

	roleID := plan.RoleId
	if roleID == "" {
		roleID = DefaultPlanRoleID
	}

	var roleId pgtype.UUID
	if err := roleId.Scan(roleID); err != nil {
		return "", fmt.Errorf("invalid roleId %q: %w", roleID, err)
	}

	err = store.CreatePlan(ctx, sqlc.CreatePlanParams{
		Name:            plan.Name,
		Description:     pgtype.Text{String: plan.Description, Valid: plan.Description != ""},
		StripePriceID:   pgtype.Text{String: plan.StripePriceID, Valid: plan.StripePriceID != ""},
		StripeProductID: plan.StripeProductID,
		Price:           int32(plan.Price),
		Active:          plan.Active,
		RoleId:          roleId,
	})
	if err != nil {
		if isDuplicateKey(err) {
			return MergePlan(ctx, store, plan)
		}
		return "", err
	}

	return "Plan created successfully", nil
}

// MergePlan atualiza um plano já existente preservando os campos que o
// evento não trouxe.
func MergePlan(ctx context.Context, store stripeWebhookStore, plan Plan) (string, error) {
	existingProductID := plan.StripeProductID
	existingName := ""
	var existingDescription pgtype.Text
	var existingPrice int32
	var existingPriceID pgtype.Text
	var existingActive bool

	byProduct, err := store.GetPlanByStripeProductID(ctx, plan.StripeProductID)
	if err == nil {
		existingName = byProduct.Name
		existingDescription = byProduct.Description
		existingPrice = byProduct.Price
		existingPriceID = byProduct.StripePriceID
		existingActive = byProduct.Active
	} else if errors.Is(err, pgx.ErrNoRows) && plan.StripePriceID != "" {
		byPrice, priceErr := store.GetPlanByPriceId(ctx, pgtype.Text{String: plan.StripePriceID, Valid: true})
		if priceErr != nil {
			if errors.Is(priceErr, pgx.ErrNoRows) {
				// Estado do banco não permite o merge; reprocessar o mesmo
				// evento produziria o mesmo resultado — não retenir o webhook.
				log.Printf("merge plan: product %s / price %s não encontrado", plan.StripeProductID, plan.StripePriceID)
				return "Plan not found; nothing to merge.", nil
			}
			return "", priceErr
		}
		existingProductID = byPrice.StripeProductID
		existingName = byPrice.Name
		existingDescription = byPrice.Description
		existingPrice = byPrice.Price
		existingPriceID = byPrice.StripePriceID
		existingActive = byPrice.Active
	} else if errors.Is(err, pgx.ErrNoRows) {
		log.Printf("merge plan: product %s não encontrado", plan.StripeProductID)
		return "Plan not found; nothing to merge.", nil
	} else {
		return "", err
	}

	stripePriceID := plan.StripePriceID
	if stripePriceID == "" {
		stripePriceID = existingPriceID.String
	}

	price := plan.Price
	if price == 0 {
		price = int64(existingPrice)
	}

	description := plan.Description
	if description == "" {
		description = existingDescription.String
	}

	name := plan.Name
	if name == "" {
		name = existingName
	}

	active := plan.Active
	if plan.DeactivateOnly && plan.Active {
		active = existingActive
	}

	err = store.UpdatePlan(ctx, sqlc.UpdatePlanParams{
		StripeProductID: existingProductID,
		StripePriceID:   pgtype.Text{String: stripePriceID, Valid: stripePriceID != ""},
		Name:            name,
		Description:     pgtype.Text{String: description, Valid: description != ""},
		Price:           int32(price),
		Active:          active,
		UpdatedAt:       pgtype.Timestamp{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return "", err
	}

	return "Plan updated successfully", nil
}

// DeactivatePlan desativa um plano (soft delete lógico via Active=false).
func DeactivatePlan(ctx context.Context, store stripeWebhookStore, plan Plan) (string, error) {
	exists, err := store.ExistsPlanByProductId(ctx, plan.StripeProductID)
	if err != nil {
		return "", err
	}

	if !exists {
		return "Plan not found, nothing to deactivate", nil
	}

	err = store.DeactivatePlan(ctx, sqlc.DeactivatePlanParams{
		StripeProductID: plan.StripeProductID,
		UpdatedAt:       pgtype.Timestamp{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return "", err
	}

	return "Plan deactivated successfully", nil
}

func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func GetPlans(c *echo.Context, db *database.Database) error {
	plans, err := db.Query.GetPlans(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Mapeia para DTO para emitir o preço em reais (o banco guarda centavos).
	// Mantém as chaves PascalCase (Id/Name/Description/Price) do contrato público.
	result := make([]dto.PublicPlanResponse, 0, len(plans))
	for _, plan := range plans {
		result = append(result, dto.MapPublicPlanResponse(plan))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Plans listed successfully", "data": result})
}

// GetAdminPlans lista TODOS os planos (ativos e inativos) com campos completos
// em camelCase, para o painel admin.
func GetAdminPlans(c *echo.Context, db *database.Database) error {
	plans, err := db.Query.GetAllPlans(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	result := make([]dto.PlanResponse, 0, len(plans))
	for _, plan := range plans {
		result = append(result, dto.MapPlanResponse(plan))
	}

	// map (e não dto.ResponseBase) para garantir que listas vazias sejam
	// emitidas como "data": [] — ResponseBase usa omitempty e omitiria o campo.
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Plans listed successfully",
		"data":    result,
	})
}

func UpdatePlan(c *echo.Context, db *database.Database, plan Plan) error {
	exists, err := db.Query.ExistsPlanByProductId(c.Request().Context(), plan.StripeProductID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Plan not found"})
	}

	existingPlan, err := db.Query.GetPlanByStripeProductID(c.Request().Context(), plan.StripeProductID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	name := plan.Name
	if name == "" {
		name = existingPlan.Name
	}

	description := plan.Description
	if description == "" && existingPlan.Description.Valid {
		description = existingPlan.Description.String
	}

	price := plan.Price
	if price == 0 && existingPlan.Price != 0 {
		price = int64(existingPlan.Price)
	}

	stripePriceID := plan.StripePriceID
	if stripePriceID == "" && existingPlan.StripePriceID.Valid {
		stripePriceID = existingPlan.StripePriceID.String
	}

	err = db.Query.UpdatePlan(c.Request().Context(), sqlc.UpdatePlanParams{
		StripeProductID: plan.StripeProductID,
		StripePriceID:   pgtype.Text{String: stripePriceID, Valid: stripePriceID != ""},
		Name:            name,
		Description:     pgtype.Text{String: description, Valid: description != ""},
		Price:           int32(price),
		Active:          plan.Active,
		UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Plan updated successfully"})
}
