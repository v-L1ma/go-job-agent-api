package dto

import (
	sqlc "job-agent-api/internal/queries"
)

type MySubscriptionResponse struct {
	Id                string       `json:"id"`
	PlanId            string       `json:"planId"`
	Status            string       `json:"status"`
	CancelAtPeriodEnd bool         `json:"cancelAtPeriodEnd"`
	CurrentPeriodEnd  *string      `json:"currentPeriodEnd"`
	Plan              *PlanSummary `json:"plan"`
}

// PlanSummary resume o plano na assinatura do usuário. O banco guarda o
// preço em centavos (int); aqui ele é emitido em reais (float).
type PlanSummary struct {
	Id          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

// AdminSubscriptionResponse representa uma assinatura para o painel admin
// (camelCase). `UserEmail` e `RoleName` (nome do cargo do plano) vêm dos
// LEFT JOINs (vazios quando usuário/plano/cargo não existe). Datas opcionais
// são null quando ausentes/inválidas.
type AdminSubscriptionResponse struct {
	Id                      string  `json:"id"`
	UserEmail               string  `json:"userEmail"`
	RoleName                string  `json:"roleName"`
	Plan                    string  `json:"plan"`
	StripeCustomerId        string  `json:"stripeCustomerId"`
	StripeSubscriptionId    string  `json:"stripeSubscriptionId"`
	StripeCheckoutSessionId string  `json:"stripeCheckoutSessionId"`
	StripePriceId           string  `json:"stripePriceId"`
	Status                  string  `json:"status"`
	CurrentPeriodStart      *string `json:"currentPeriodStart"`
	CurrentPeriodEnd        *string `json:"currentPeriodEnd"`
	CancelAtPeriodEnd       bool    `json:"cancelAtPeriodEnd"`
	CanceledAt              *string `json:"canceledAt"`
	CreatedAt               string  `json:"createdAt"`
	UpdatedAt               string  `json:"updatedAt"`
}

func MapAdminSubscriptionResponse(sub sqlc.GetSubscriptionsRow) AdminSubscriptionResponse {
	response := AdminSubscriptionResponse{
		Id:                      sub.Id.String(),
		UserEmail:               sub.UserEmail.String,
		RoleName:                sub.PlanRoleName.String,
		Plan:                    sub.PlanName.String,
		StripeCustomerId:        sub.StripeCustomerID,
		StripeSubscriptionId:    sub.StripeSubscriptionID,
		StripeCheckoutSessionId: sub.StripeCheckoutSessionID.String,
		StripePriceId:           sub.StripePriceID.String,
		Status:                  sub.Status,
		CancelAtPeriodEnd:       sub.CancelAtPeriodEnd,
		CreatedAt:               timestampToRFC3339(sub.CreatedAt),
		UpdatedAt:               timestampToRFC3339(sub.UpdatedAt),
	}

	if sub.CurrentPeriodStart.Valid {
		periodStart := sub.CurrentPeriodStart.Time.Format("2006-01-02T15:04:05Z07:00")
		response.CurrentPeriodStart = &periodStart
	}
	if sub.CurrentPeriodEnd.Valid {
		periodEnd := sub.CurrentPeriodEnd.Time.Format("2006-01-02T15:04:05Z07:00")
		response.CurrentPeriodEnd = &periodEnd
	}
	if sub.CanceledAt.Valid {
		canceledAt := sub.CanceledAt.Time.Format("2006-01-02T15:04:05Z07:00")
		response.CanceledAt = &canceledAt
	}

	return response
}

// MapAdminSubscriptionResponseFromSubscription converte o retorno de
// GetSubscriptionById para o MESMO shape de AdminSubscriptionResponse da
// lista — email do usuário, nome do plano e nome do cargo já vêm do LEFT JOIN
// da query, então nenhum campo precisa ser preenchido pelo handler.
func MapAdminSubscriptionResponseFromSubscription(sub sqlc.GetSubscriptionByIdRow) AdminSubscriptionResponse {
	return MapAdminSubscriptionResponse(sqlc.GetSubscriptionsRow{
		Id:                      sub.Id,
		UserEmail:               sub.UserEmail,
		PlanId:                  sub.PlanId,
		StripeCustomerID:        sub.StripeCustomerID,
		StripeSubscriptionID:    sub.StripeSubscriptionID,
		StripeCheckoutSessionID: sub.StripeCheckoutSessionID,
		StripePriceID:           sub.StripePriceID,
		Status:                  sub.Status,
		CurrentPeriodStart:      sub.CurrentPeriodStart,
		CurrentPeriodEnd:        sub.CurrentPeriodEnd,
		CancelAtPeriodEnd:       sub.CancelAtPeriodEnd,
		CanceledAt:              sub.CanceledAt,
		CreatedAt:               sub.CreatedAt,
		UpdatedAt:               sub.UpdatedAt,
		PlanName:                sub.PlanName,
		PlanRoleName:            sub.PlanRoleName,
	})
}

// MapAdminSubscriptionResponseFromGetSubscriptionsByUserId converte o retorno
// de GetSubscriptionsByUserId para o MESMO shape de AdminSubscriptionResponse
// da lista (o sqlc gera um row type distinto por query, embora as colunas
// sejam idênticas — a cópia abaixo é campos a campos e delega ao mapper
// canônico, mantendo a lógica de datas/conversão em um único lugar).
func MapAdminSubscriptionResponseFromGetSubscriptionsByUserId(sub sqlc.GetSubscriptionsByUserIdRow) AdminSubscriptionResponse {
	return MapAdminSubscriptionResponse(sqlc.GetSubscriptionsRow{
		Id:                      sub.Id,
		UserEmail:               sub.UserEmail,
		PlanId:                  sub.PlanId,
		StripeCustomerID:        sub.StripeCustomerID,
		StripeSubscriptionID:    sub.StripeSubscriptionID,
		StripeCheckoutSessionID: sub.StripeCheckoutSessionID,
		StripePriceID:           sub.StripePriceID,
		Status:                  sub.Status,
		CurrentPeriodStart:      sub.CurrentPeriodStart,
		CurrentPeriodEnd:        sub.CurrentPeriodEnd,
		CancelAtPeriodEnd:       sub.CancelAtPeriodEnd,
		CanceledAt:              sub.CanceledAt,
		CreatedAt:               sub.CreatedAt,
		UpdatedAt:               sub.UpdatedAt,
		PlanName:                sub.PlanName,
		PlanRoleName:            sub.PlanRoleName,
	})
}

// AdminUserResponse resume o usuário no painel admin (camelCase).
//
// Segurança: expõe APENAS id, name, email e role. PasswordHash e qualquer
// outro campo sensível de AspNetUsers NUNCA entram neste DTO.
// email é "" quando NULL no banco.
type AdminUserResponse struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// MapAdminUserResponse converte a linha de GetUserById para o DTO admin,
// descartando todos os campos sensíveis (PasswordHash etc.).
func MapAdminUserResponse(user sqlc.GetUserByIdRow) AdminUserResponse {
	return AdminUserResponse{
		Id:    user.Id.String(),
		Name:  user.Name,
		Email: user.Email.String, // pgtype.Text: "" quando NULL
		Role:  user.RoleName,     // COALESCE(..., 'Standard') na query
	}
}

// SubscriptionDetailResponse é o payload de data do detalhe de assinatura
// (GET /admin/subscriptions/:subscriptionId).
//
// Nullability: plan e user podem ser null (defensivo; user não deveria ser
// null por causa da FK). invoices é sempre uma lista — `[]` quando não há
// faturas (nunca null/ausente).
type SubscriptionDetailResponse struct {
	Subscription AdminSubscriptionResponse `json:"subscription"`
	Plan         *PlanResponse             `json:"plan"`
	User         *AdminUserResponse        `json:"user"`
	Invoices     []InvoiceResponse         `json:"invoices"`
}

// UserSubscriptionsResponse é o payload de data de
// GET /admin/users/:userId/subscriptions. subscriptions é sempre uma lista
// (`[]` quando vazia, nunca null/ausente).
type UserSubscriptionsResponse struct {
	User          AdminUserResponse           `json:"user"`
	Subscriptions []AdminSubscriptionResponse `json:"subscriptions"`
}
