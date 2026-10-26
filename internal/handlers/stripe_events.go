package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"
)

// parseStripeEvent valida o header Stripe-Signature contra o payload e
// parseia o evento. Retorna erro quando a assinatura é inválida (ou o secret
// não está configurado); nesse caso o evento NÃO deve ser processado.
//
// IgnoreAPIVersionMismatch: a versão de API do endpoint de webhook é
// independente da versão do SDK — sem essa opção, eventos com api_version de
// um "release train" diferente seriam rejeitados mesmo com assinatura
// válida. A validação criptográfica da assinatura e a tolerância de
// timestamp (300s) continuam ativas.
func parseStripeEvent(payload []byte, signature, secret string) (stripe.Event, error) {
	return webhook.ConstructEventWithOptions(payload, signature, secret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
}

// ---------------------------------------------------------------------------
// Extractors
//
// Funções puras (JSON -> struct) usadas pelos handlers de webhook. Permitem
// testar o parsing isoladamente, sem banco e sem rede.
// ---------------------------------------------------------------------------

// checkoutEventData são os campos relevantes de `checkout.session.completed`.
type checkoutEventData struct {
	SessionID         string
	CustomerID        string
	SubscriptionID    string
	ClientReferenceID string
	// UserID vem de metadata[user_id]; se ausente, cai para client_reference_id.
	// NUNCA usamos e-mail para identificar o usuário.
	UserID string
	// PlanID vem de metadata[plan_id].
	PlanID string
	Status string
}

func extractCheckoutEventData(raw []byte) (*checkoutEventData, error) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, fmt.Errorf("parse checkout session: %w", err)
	}

	info := &checkoutEventData{
		SessionID:         session.ID,
		ClientReferenceID: session.ClientReferenceID,
		Status:            string(session.Status),
		UserID:            session.Metadata["user_id"],
		PlanID:            session.Metadata["plan_id"],
	}
	if session.Customer != nil {
		info.CustomerID = session.Customer.ID
	}
	if session.Subscription != nil {
		info.SubscriptionID = session.Subscription.ID
	}
	if info.UserID == "" {
		info.UserID = session.ClientReferenceID
	}
	return info, nil
}

// subscriptionEventData são os campos relevantes de
// `customer.subscription.created|updated|deleted`.
type subscriptionEventData struct {
	ID         string
	CustomerID string
	Status     string
	PriceID    string
	// CurrentPeriodStart/End em unix seconds (0 = desconhecido).
	CurrentPeriodStart int64
	CurrentPeriodEnd   int64
	CancelAtPeriodEnd  bool
	// CanceledAt em unix seconds (0 = não cancelada).
	CanceledAt int64
	// UserID/PlanID vêm da metadata do objeto Subscription.
	UserID string
	PlanID string
}

func extractSubscriptionEventData(raw []byte) (*subscriptionEventData, error) {
	// Os campos current_period_start/end migraram de Subscription para
	// SubscriptionItem em versões mais novas da API. Suportamos os dois
	// formatos: o objeto Subscription (itens/price/status/metadata) e os
	// campos de período antigos, quando presentes no próprio objeto.
	//
	// IMPORTANTE: os campos legados são decodificados em um struct separado
	// porque stripe.Subscription possui UnmarshalJSON próprio — se embutido
	// aqui, ele se tornaria o UnmarshalJSON do struct anônimo (promoção de
	// métodos) e ignoraria os campos do nível externo.
	var sub stripe.Subscription
	if err := json.Unmarshal(raw, &sub); err != nil {
		return nil, fmt.Errorf("parse subscription: %w", err)
	}
	var legacy struct {
		CurrentPeriodStart int64 `json:"current_period_start"`
		CurrentPeriodEnd   int64 `json:"current_period_end"`
	}
	_ = json.Unmarshal(raw, &legacy) // melhor esforço: formato antigo

	info := &subscriptionEventData{
		ID:                sub.ID,
		Status:            string(sub.Status),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
		CanceledAt:        sub.CanceledAt,
	}
	if sub.Customer != nil {
		info.CustomerID = sub.Customer.ID
	}
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		if item != nil {
			if item.Price != nil {
				info.PriceID = item.Price.ID
			}
			info.CurrentPeriodStart = item.CurrentPeriodStart
			info.CurrentPeriodEnd = item.CurrentPeriodEnd
		}
	}
	if info.CurrentPeriodStart == 0 {
		info.CurrentPeriodStart = legacy.CurrentPeriodStart
	}
	if info.CurrentPeriodEnd == 0 {
		info.CurrentPeriodEnd = legacy.CurrentPeriodEnd
	}
	info.UserID = sub.Metadata["user_id"]
	info.PlanID = sub.Metadata["plan_id"]
	return info, nil
}

// invoiceEventData são os campos relevantes de
// `invoice.paid|invoice.payment_failed`.
type invoiceEventData struct {
	ID             string
	SubscriptionID string
	CustomerID     string
	Currency       string
	AmountPaid     int64
	Created        int64
	// Status é o status do objeto Invoice do Stripe ("paid", "open", ...).
	// O status gravado na nossa tabela é definido pelo tipo do evento.
	Status string
}

func extractInvoiceEventData(raw []byte) (*invoiceEventData, error) {
	// A referência `invoice.subscription` foi substituída por
	// `invoice.parent.subscription_details.subscription` em versões novas da
	// API; aceitamos os dois formatos.
	var payload struct {
		ID           string                `json:"id"`
		Status       string                `json:"status"`
		Currency     string                `json:"currency"`
		AmountPaid   int64                 `json:"amount_paid"`
		Created      int64                 `json:"created"`
		Customer     *stripe.Customer      `json:"customer"`
		Subscription *stripe.Subscription  `json:"subscription"` // formato antigo
		Parent       *stripe.InvoiceParent `json:"parent"`       // formato novo
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse invoice: %w", err)
	}

	info := &invoiceEventData{
		ID:         payload.ID,
		Currency:   payload.Currency,
		AmountPaid: payload.AmountPaid,
		Created:    payload.Created,
		Status:     payload.Status,
	}
	if payload.Customer != nil {
		info.CustomerID = payload.Customer.ID
	}
	switch {
	case payload.Parent != nil && payload.Parent.SubscriptionDetails != nil && payload.Parent.SubscriptionDetails.Subscription != nil:
		info.SubscriptionID = payload.Parent.SubscriptionDetails.Subscription.ID
	case payload.Subscription != nil:
		info.SubscriptionID = payload.Subscription.ID
	}
	return info, nil
}

// ---------------------------------------------------------------------------
// Helpers de conversão
// ---------------------------------------------------------------------------

// parseUUID converte string em pgtype.UUID; retorna UUID inválido para
// strings vazias ou malformadas (sem retornar erro — o chamador decide o que
// fazer com a falha de resolução).
func parseUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if s == "" {
		return pgtype.UUID{}
	}
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}
	}
	return u
}

// timestampFromUnix converte unix seconds em pgtype.Timestamp (NULL quando
// sec <= 0).
func timestampFromUnix(sec int64) pgtype.Timestamp {
	if sec <= 0 {
		return pgtype.Timestamp{}
	}
	return pgtype.Timestamp{Time: time.Unix(sec, 0).UTC(), Valid: true}
}
