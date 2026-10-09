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
type PlanResponse struct {
	Id            string  `json:"id"`
	Name          string  `json:"name"`
	StripePriceId string  `json:"stripePriceId"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	RoleId        string  `json:"roleId"`
	Active        bool    `json:"active"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

// MapPlanResponse converte a linha do sqlc para o DTO admin, com o preço
// convertido de centavos para reais (somente leitura; a escrita continua
// em centavos).
func MapPlanResponse(plan sqlc.GetAllPlansRow) PlanResponse {
	return PlanResponse{
		Id:            plan.Id.String(),
		Name:          plan.Name,
		StripePriceId: plan.StripePriceID.String,
		Description:   plan.Description.String,
		Price:         CentsToBRL(plan.Price),
		RoleId:        plan.RoleId.String(),
		Active:        plan.Active,
		CreatedAt:     timestampToRFC3339(plan.CreatedAt),
		UpdatedAt:     timestampToRFC3339(plan.UpdatedAt),
	}
}

// MapPlanResponseFromGetPlanById converte a linha de GetPlanById para o
// MESMO shape JSON de PlanResponse (id, name, description, stripePriceId,
// price, roleId, active, createdAt, updatedAt), com o preço convertido de
// centavos para reais. Usado pelo detalhe de assinatura do admin.
func MapPlanResponseFromGetPlanById(plan sqlc.GetPlanByIdRow) PlanResponse {
	return PlanResponse{
		Id:            plan.Id.String(),
		Name:          plan.Name,
		StripePriceId: plan.StripePriceID.String,
		Description:   plan.Description.String,
		Price:         CentsToBRL(plan.Price),
		RoleId:        plan.RoleId.String(),
		Active:        plan.Active,
		CreatedAt:     timestampToRFC3339(plan.CreatedAt),
		UpdatedAt:     timestampToRFC3339(plan.UpdatedAt),
	}
}
