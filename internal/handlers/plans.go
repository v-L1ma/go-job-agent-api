package handlers

import (
	"errors"
	"net/http"
	"time"

	"job-agent-api/internal/database"
	sqlc "job-agent-api/internal/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

const DefaultPlanRoleID = "a2238759-645f-4d7e-867e-1ccf3d86afef"

type Plan struct {
	Name          string
	Description   string
	StripePriceID string
	StripeProductID string
	Price         int64
	Active        bool
	RoleId        string
	// DeactivateOnly indica que o evento so pode desativar o plano,
	// nunca reativar (ex.: eventos de price nao devem reativar um plano
	// desativado pelo produto).
	DeactivateOnly bool
}

func CreatePlan(c *echo.Context, db *database.Database, plan Plan) error {		
	alreadyExists, err := db.Query.ExistsPlanByProductId(c.Request().Context(), plan.StripeProductID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if alreadyExists {
		return MergePlan(c, db, plan)
	}

	roleID := plan.RoleId
	if roleID == "" {
		roleID = DefaultPlanRoleID
	}

	var roleId pgtype.UUID
	err = roleId.Scan(roleID)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid roleId parameter",
		})
	}

	err = db.Query.CreatePlan(c.Request().Context(), sqlc.CreatePlanParams{
		Name:          plan.Name,
		Description:   pgtype.Text{String: plan.Description, Valid: plan.Description != ""},
		StripePriceID: pgtype.Text{String: plan.StripePriceID, Valid: plan.StripePriceID != ""},
		StripeProductID:  plan.StripeProductID,
		Price:         int32(plan.Price),
		Active:        plan.Active,
		RoleId:        roleId,
	})
	if err != nil {
		if isDuplicateKey(err) {
			return MergePlan(c, db, plan)
		}
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Plan created successfully"})
}

func MergePlan(c *echo.Context, db *database.Database, plan Plan) error {
	ctx := c.Request().Context()

	existingProductID := plan.StripeProductID
	existingName := ""
	var existingDescription pgtype.Text
	var existingPrice int32
	var existingPriceID pgtype.Text
	var existingActive bool

	byProduct, err := db.Query.GetPlanByStripeProductID(ctx, plan.StripeProductID)
	if err == nil {
		existingName = byProduct.Name
		existingDescription = byProduct.Description
		existingPrice = byProduct.Price
		existingPriceID = byProduct.StripePriceID
		existingActive = byProduct.Active
	} else if errors.Is(err, pgx.ErrNoRows) && plan.StripePriceID != "" {
		byPrice, priceErr := db.Query.GetPlanByPriceId(ctx, pgtype.Text{String: plan.StripePriceID, Valid: true})
		if priceErr != nil {
			if errors.Is(priceErr, pgx.ErrNoRows) {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "Plan not found"})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": priceErr.Error()})
		}
		existingProductID = byPrice.StripeProductID
		existingName = byPrice.Name
		existingDescription = byPrice.Description
		existingPrice = byPrice.Price
		existingPriceID = byPrice.StripePriceID
		existingActive = byPrice.Active
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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

	err = db.Query.UpdatePlan(ctx, sqlc.UpdatePlanParams{
		StripeProductID: existingProductID,
		StripePriceID:   pgtype.Text{String: stripePriceID, Valid: stripePriceID != ""},
		Name:            name,
		Description:     pgtype.Text{String: description, Valid: description != ""},
		Price:           int32(price),
		Active:          active,
		UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Plan updated successfully"})
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

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Plan created successfully", "data": plans})
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

func DeactivatePlan(c *echo.Context, db *database.Database, plan Plan) error {

	exists, err := db.Query.ExistsPlanByProductId(c.Request().Context(), plan.StripeProductID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if (!exists) {
		return c.JSON(http.StatusOK, map[string]string{"message": "Plan not found, nothing to deactivate"})
	}

	err = db.Query.DeactivatePlan(c.Request().Context(), sqlc.DeactivatePlanParams{
		StripeProductID: plan.StripeProductID,
		UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Plan deactivated successfully"})
}

