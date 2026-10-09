package handlers

import (
	"context"

	sqlc "job-agent-api/internal/queries"

	"github.com/jackc/pgx/v5/pgtype"
)

// stripeWebhookStore agrupa as queries sqlc usadas pelo dispatcher de
// webhooks do Stripe.
//
// Em produção a implementação é *sqlc.Queries (diretamente ou
// *sqlc.Queries.WithTx(tx) para rodar dentro da transação do evento). Os
// testes usam uma implementação em memória — o que permite exercitar os
// fluxos completos (checkout, criação, renovação, upgrade, cancelamento,
// reenvio e ordem fora de sequência) sem banco de dados nem rede.
type stripeWebhookStore interface {
	// Idempotência
	TryInsertWebhookEvent(ctx context.Context, arg sqlc.TryInsertWebhookEventParams) (string, error)

	// Stripe Customers (usuário <-> customer)
	UpsertStripeCustomer(ctx context.Context, arg sqlc.UpsertStripeCustomerParams) error
	GetUserIdByStripeCustomerId(ctx context.Context, stripecustomerid string) (pgtype.UUID, error)

	// Assinaturas
	SetSubscriptionCheckoutSession(ctx context.Context, arg sqlc.SetSubscriptionCheckoutSessionParams) (int64, error)
	InsertSubscriptionFromCheckout(ctx context.Context, arg sqlc.InsertSubscriptionFromCheckoutParams) (int64, error)
	UpsertSubscription(ctx context.Context, arg sqlc.UpsertSubscriptionParams) (pgtype.UUID, error)
	GetSubscriptionByStripeSubscriptionId(ctx context.Context, stripesubscriptionID string) (sqlc.Subscription, error)

	// Planos (resolução de plano/role para a assinatura)
	GetPlanByPriceId(ctx context.Context, stripePriceID pgtype.Text) (sqlc.GetPlanByPriceIdRow, error)
	GetPlanRoleId(ctx context.Context, id pgtype.UUID) (pgtype.UUID, error)

	// Papéis (concessão/revogação de acesso)
	AssignRoleToUser(ctx context.Context, arg sqlc.AssignRoleToUserParams) error
	RemoveRoleFromUser(ctx context.Context, arg sqlc.RemoveRoleFromUserParams) error

	// Cobranças
	UpsertStripeInvoice(ctx context.Context, arg sqlc.UpsertStripeInvoiceParams) error

	// Sincronização de products/prices -> Plans (webhooks product.*/price.*)
	ExistsPlanByProductId(ctx context.Context, stripeproductID string) (bool, error)
	GetPlanByStripeProductID(ctx context.Context, stripeproductID string) (sqlc.GetPlanByStripeProductIDRow, error)
	CreatePlan(ctx context.Context, arg sqlc.CreatePlanParams) error
	UpdatePlan(ctx context.Context, arg sqlc.UpdatePlanParams) error
	DeactivatePlan(ctx context.Context, arg sqlc.DeactivatePlanParams) error
}
