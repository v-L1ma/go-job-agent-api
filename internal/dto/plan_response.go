package dto

import (
	"time"

	sqlc "job-agent-api/internal/queries"

	"github.com/jackc/pgx/v5/pgtype"
)

// timestampToRFC3339 converte um pgtype.Timestamp para string RFC3339.
// Retorna string vazia quando o timestamp é inválido (NULL no banco).
func timestampToRFC3339(ts pgtype.Timestamp) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.Format(time.RFC3339)
}

// PublicPlanResponse representa um plano do endpoint público GET /plans.
// As chaves JSON são PascalCase por contrato público (já documentado e
// consumido pelo frontend). O banco guarda o preço em centavos (int); aqui
// ele é emitido em reais (float).
type PublicPlanResponse struct {
	Id          string  `json:"Id"`
	Name        string  `json:"Name"`
	Description string  `json:"Description"`
	Price       float64 `json:"Price"`
}

// MapPublicPlanResponse converte uma linha crua do sqlc (GetPlansRow) para o
// contrato público, com o preço convertido de centavos para reais.
// Description é "" quando NULL no banco (pgtype.Text inválido).
func MapPublicPlanResponse(plan sqlc.GetPlansRow) PublicPlanResponse {
	return PublicPlanResponse{
		Id:          plan.Id.String(),
		Name:        plan.Name,
		Description: plan.Description.String,
		Price:       CentsToBRL(plan.Price),
	}
}

// PlanResponse representa um plano para o painel admin (camelCase).
// O banco guarda o preço em centavos (int); aqui ele é emitido em reais (float).
// Os limites diários são nullable no banco: nil = ilimitado.
type PlanResponse struct {
	Id                    string  `json:"id"`
	Name                  string  `json:"name"`
	StripePriceId         string  `json:"stripePriceId"`
	Description           string  `json:"description"`
	Price                 float64 `json:"price"`
	RoleId                string  `json:"roleId"`
	Active                bool    `json:"active"`
	DailyCvLimit          *int32  `json:"dailyCvLimit"`
	DailyApplicationLimit *int32  `json:"dailyApplicationLimit"`
	CreatedAt             string  `json:"createdAt"`
	UpdatedAt             string  `json:"updatedAt"`
}

// int4ToPtr converte um pgtype.Int4 nullable para *int32 (nil = NULL no banco).
func int4ToPtr(v pgtype.Int4) *int32 {
	if !v.Valid {
		return nil
	}
	value := v.Int32
	return &value
}

// MapPlanResponse converte a linha do sqlc para o DTO admin, com o preço
// convertido de centavos para reais (somente leitura; a escrita continua
// em centavos).
func MapPlanResponse(plan sqlc.GetAllPlansRow) PlanResponse {
	return PlanResponse{
		Id:                    plan.Id.String(),
		Name:                  plan.Name,
		StripePriceId:         plan.StripePriceID.String,
		Description:           plan.Description.String,
		Price:                 CentsToBRL(plan.Price),
		RoleId:                plan.RoleId.String(),
		Active:                plan.Active,
		DailyCvLimit:          int4ToPtr(plan.DailyCvLimit),
		DailyApplicationLimit: int4ToPtr(plan.DailyApplicationLimit),
		CreatedAt:             timestampToRFC3339(plan.CreatedAt),
		UpdatedAt:             timestampToRFC3339(plan.UpdatedAt),
	}
}

// MapPlanResponseFromGetPlanById converte a linha de GetPlanById para o
// MESMO shape JSON de PlanResponse (id, name, description, stripePriceId,
// price, roleId, active, dailyCvLimit, dailyApplicationLimit, createdAt,
// updatedAt), com o preço convertido de centavos para reais. Usado pelo
// detalhe de assinatura do admin.
func MapPlanResponseFromGetPlanById(plan sqlc.GetPlanByIdRow) PlanResponse {
	return PlanResponse{
		Id:                    plan.Id.String(),
		Name:                  plan.Name,
		StripePriceId:         plan.StripePriceID.String,
		Description:           plan.Description.String,
		Price:                 CentsToBRL(plan.Price),
		RoleId:                plan.RoleId.String(),
		Active:                plan.Active,
		DailyCvLimit:          int4ToPtr(plan.DailyCvLimit),
		DailyApplicationLimit: int4ToPtr(plan.DailyApplicationLimit),
		CreatedAt:             timestampToRFC3339(plan.CreatedAt),
		UpdatedAt:             timestampToRFC3339(plan.UpdatedAt),
	}
}

// PlanLimits representa os limites diários de um plano no admin.
// Campos sempre emitidos (sem omitempty): nil = ilimitado → JSON null.
type PlanLimits struct {
	DailyCvLimit          *int32 `json:"dailyCvLimit"`
	DailyApplicationLimit *int32 `json:"dailyApplicationLimit"`
}

// NewPlanLimits monta o payload de limites a partir dos valores nullable do
// banco (pgtype.Int4).
func NewPlanLimits(cvLimit, applicationLimit pgtype.Int4) PlanLimits {
	return PlanLimits{
		DailyCvLimit:          int4ToPtr(cvLimit),
		DailyApplicationLimit: int4ToPtr(applicationLimit),
	}
}
