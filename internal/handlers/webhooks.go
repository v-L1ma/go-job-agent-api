package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"job-agent-api/internal/database"
	sqlc "job-agent-api/internal/queries"
	"job-agent-api/internal/services"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/stripe/stripe-go/v86"
)

// StripeWebhook processa os eventos do Stripe.
//
// Fluxo (fonte da verdade = webhooks):
//
//  1. Valida Stripe-Signature          → inválida: HTTP 400 (não processa)
//  2. BEGIN + claim atômico do event.id → já existe: COMMIT e HTTP 200
//  3. Handler do evento na MESMA tx     → erro: ROLLBACK e HTTP 500 (Stripe reenvia)
//  4. COMMIT                            → HTTP 200
//
// O ROLLBACK desfaz também o claim de idempotência, permitindo o reenvio.
func StripeWebhook(c *echo.Context, db *database.Database) error {
	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "failed to read request body",
		})
	}

	signature := c.Request().Header.Get("Stripe-Signature")
	event, err := parseStripeEvent(payload, signature, services.StripeWebhookSecret())
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid stripe signature",
		})
	}

	ctx := c.Request().Context()

	tx, err := db.Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := db.Query.WithTx(tx)
	message, alreadyProcessed, err := runWebhookEvent(ctx, queries, &event)
	if err != nil {
		log.Printf("stripe webhook [%s/%s]: %v", event.Type, event.ID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	committed = true

	if alreadyProcessed {
		return c.JSON(http.StatusOK, map[string]string{"message": message})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": message})
}

// runWebhookEvent executa o claim de idempotência e o dispatch do evento.
// Retorna alreadyProcessed=true quando o event.id já havia sido registrado
// (nenhum processamento adicional é feito).
func runWebhookEvent(ctx context.Context, store stripeWebhookStore, event *stripe.Event) (message string, alreadyProcessed bool, err error) {
	// Claim atômico: INSERT ... ON CONFLICT ("Id") DO NOTHING RETURNING "Id".
	// pgx.ErrNoRows => o event.id já existia => evento duplicado.
	if _, err := store.TryInsertWebhookEvent(ctx, sqlc.TryInsertWebhookEventParams{
		Id:   event.ID,
		Type: string(event.Type),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "Evento já processado; ignorando duplicata.", true, nil
		}
		return "", false, fmt.Errorf("claim de idempotência: %w", err)
	}

	message, err = dispatchStripeEvent(ctx, store, event)
	if err != nil {
		return "", false, err
	}
	return message, false, nil
}

// dispatchStripeEvent roteia o evento para o handler correspondente.
// Todo handler opera apenas com webhook + banco (nenhuma chamada obrigatória
// à API do Stripe), o que mantém o processamento hermético e resiliente à
// ordem de entrega.
func dispatchStripeEvent(ctx context.Context, store stripeWebhookStore, event *stripe.Event) (string, error) {
	raw := eventPayload(event)

	switch event.Type {
	case "checkout.session.completed":
		return handleCheckoutCompleted(ctx, store, raw)

	case "customer.subscription.created":
		return handleSubscriptionCreated(ctx, store, raw)

	case "customer.subscription.updated":
		return handleSubscriptionUpdated(ctx, store, raw)

	case "customer.subscription.deleted":
		return handleSubscriptionDeleted(ctx, store, raw)

	case "invoice.paid":
		return handleInvoiceEvent(ctx, store, raw, "paid")

	case "invoice.payment_failed":
		return handleInvoiceEvent(ctx, store, raw, "payment_failed")

	case "price.created", "price.updated":
		return handlePriceSyncEvent(ctx, store, raw)

	case "product.created", "product.updated":
		return handleProductSyncEvent(ctx, store, raw)

	case "product.deleted":
		return handleProductDeletedEvent(ctx, store, raw)

	default:
		// Evento não utilizado pela aplicação — ex.:
		//   payment_intent.succeeded, payment_intent.payment_failed,
		//   payment_method.attached.
		// O event.id já foi registrado no claim; nada mais a fazer.
		return "Evento não utilizado pela aplicação; ignorado.", nil
	}
}

func eventPayload(event *stripe.Event) json.RawMessage {
	if event == nil || event.Data == nil {
		return nil
	}
	return event.Data.Raw
}

// ---------------------------------------------------------------------------
// checkout.session.completed
// ---------------------------------------------------------------------------

// handleCheckoutCompleted relaciona o Checkout ao usuário e guarda o
// checkout_session_id. NUNCA ativa a assinatura: a ativação vem de
// customer.subscription.created/updated.
//
// É resiliente à ordem (spec seção 14):
//   - linha da assinatura já existe → apenas anexa o checkout (status intacto);
//   - linha não existe → insere registro parcial "incomplete" (sem acesso);
//   - corrida com customer.subscription.created → anexa após o insert.
func handleCheckoutCompleted(ctx context.Context, store stripeWebhookStore, raw []byte) (string, error) {
	info, err := extractCheckoutEventData(raw)
	if err != nil {
		return "", err
	}

	// Resolução do usuário SEM e-mail:
	// metadata[user_id] (ou client_reference_id) → StripeCustomers.
	userID := parseUUID(info.UserID)
	if !userID.Valid && info.CustomerID != "" {
		id, err := store.GetUserIdByStripeCustomerId(ctx, info.CustomerID)
		if err == nil {
			userID = id
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}

	// Persiste o mapeamento usuário <-> Stripe Customer.
	if userID.Valid && info.CustomerID != "" {
		if err := store.UpsertStripeCustomer(ctx, sqlc.UpsertStripeCustomerParams{
			UserId:           userID,
			StripeCustomerId: info.CustomerID,
		}); err != nil {
			return "", err
		}
	}

	if info.SubscriptionID == "" {
		log.Printf("checkout.session.completed %s sem subscription id; ignorando", info.SessionID)
		return "Checkout sem assinatura associada; ignorado.", nil
	}

	if !userID.Valid {
		// Checkout sem a metadata da aplicação (ex.: criado manualmente no
		// dashboard). Não falhar o webhook: o evento
		// customer.subscription.created trará a assinatura.
		log.Printf("checkout.session.completed %s sem user_id resolvível; ignorando", info.SessionID)
		return "Checkout sem identificação de usuário; ignorado.", nil
	}

	sessionID := pgtype.Text{String: info.SessionID, Valid: info.SessionID != ""}
	customerID := pgtype.Text{String: info.CustomerID, Valid: info.CustomerID != ""}

	// Caminho 1: a linha da assinatura já existe → anexa o checkout.
	rows, err := store.SetSubscriptionCheckoutSession(ctx, sqlc.SetSubscriptionCheckoutSessionParams{
		StripeSubscriptionID:    info.SubscriptionID,
		StripeCheckoutSessionID: sessionID,
		CustomerID:              customerID,
	})
	if err != nil {
		return "", err
	}
	if rows > 0 {
		return "Checkout vinculado à assinatura existente.", nil
	}

	// Caminho 2: linha ainda não existe → registro parcial "incomplete"
	// (HasActiveAccess("incomplete") = false, portanto sem acesso).
	planID := parseUUID(info.PlanID)
	if !planID.Valid {
		log.Printf("checkout.session.completed %s sem plan_id válido; aguardando customer.subscription.created", info.SessionID)
		return "Checkout registrado sem plano; aguardando customer.subscription.created.", nil
	}

	rows, err = store.InsertSubscriptionFromCheckout(ctx, sqlc.InsertSubscriptionFromCheckoutParams{
		UserId:                  userID,
		PlanId:                  planID,
		StripeCustomerID:        info.CustomerID,
		StripeSubscriptionID:    info.SubscriptionID,
		StripeCheckoutSessionID: sessionID,
	})
	if err != nil {
		return "", err
	}
	if rows == 0 {
		// Perdeu a corrida com customer.subscription.created: anexa o checkout
		// na linha criada pelo outro evento.
		if _, err := store.SetSubscriptionCheckoutSession(ctx, sqlc.SetSubscriptionCheckoutSessionParams{
			StripeSubscriptionID:    info.SubscriptionID,
			StripeCheckoutSessionID: sessionID,
			CustomerID:              customerID,
		}); err != nil {
			return "", err
		}
	}

	return "Checkout registrado.", nil
}

// ---------------------------------------------------------------------------
// customer.subscription.created | updated | deleted
// ---------------------------------------------------------------------------

type applySubscriptionOptions struct {
	// message retornada quando tudo ocorre bem.
	message string
	// forceCanceled impõe Status="canceled", CancelAtPeriodEnd=false e
	// CanceledAt=now (usado em customer.subscription.deleted).
	forceCanceled bool
	// tolerateUnresolved: ao invés de erro (rollback + reenvio do Stripe),
	// registra log e responde OK quando usuário/plano não são resolvíveis.
	// Usado em customer.subscription.deleted para não perder o cancelamento
	// indefinidamente.
	tolerateUnresolved bool
}

func handleSubscriptionCreated(ctx context.Context, store stripeWebhookStore, raw []byte) (string, error) {
	info, err := extractSubscriptionEventData(raw)
	if err != nil {
		return "", err
	}
	return applySubscriptionState(ctx, store, info, applySubscriptionOptions{
		message: "Assinatura criada/sincronizada.",
	})
}

func handleSubscriptionUpdated(ctx context.Context, store stripeWebhookStore, raw []byte) (string, error) {
	info, err := extractSubscriptionEventData(raw)
	if err != nil {
		return "", err
	}
	return applySubscriptionState(ctx, store, info, applySubscriptionOptions{
		message: "Assinatura sincronizada.",
	})
}

func handleSubscriptionDeleted(ctx context.Context, store stripeWebhookStore, raw []byte) (string, error) {
	info, err := extractSubscriptionEventData(raw)
	if err != nil {
		return "", err
	}
	return applySubscriptionState(ctx, store, info, applySubscriptionOptions{
		message:            "Assinatura cancelada.",
		forceCanceled:      true,
		tolerateUnresolved: true,
	})
}

// applySubscriptionState faz o UPSERT completo da assinatura e aplica a
// regra de acesso (UserRoles) dentro da mesma transação.
func applySubscriptionState(ctx context.Context, store stripeWebhookStore, info *subscriptionEventData, opts applySubscriptionOptions) (string, error) {
	if info.ID == "" {
		return "", errors.New("evento de subscription sem id")
	}

	// Linha existente (pode ter sido criada pelo checkout).
	existing, err := store.GetSubscriptionByStripeSubscriptionId(ctx, info.ID)
	hasExisting := true
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
		hasExisting = false
	}

	fail := func(reason string) (string, error) {
		if opts.tolerateUnresolved {
			log.Printf("subscription %s: %s (tolerado)", info.ID, reason)
			return "Assinatura ignorada: usuário/plano não resolvíveis.", nil
		}
		return "", errors.New(reason)
	}

	// 1. Usuário — ordem: metadata[user_id] → StripeCustomers → linha existente.
	userID := parseUUID(info.UserID)
	if !userID.Valid && info.CustomerID != "" {
		id, err := store.GetUserIdByStripeCustomerId(ctx, info.CustomerID)
		if err == nil {
			userID = id
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}
	if !userID.Valid && hasExisting {
		userID = existing.UserId
	}

	// 2. Plano — ordem: price (fonte da verdade p/ upgrade/downgrade) →
	//    metadata[plan_id] → linha existente.
	planID := pgtype.UUID{}
	if info.PriceID != "" {
		plan, err := store.GetPlanByPriceId(ctx, pgtype.Text{String: info.PriceID, Valid: true})
		if err == nil {
			planID = plan.Id
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}
	if !planID.Valid {
		planID = parseUUID(info.PlanID)
	}
	if !planID.Valid && hasExisting {
		planID = existing.PlanId
	}

	if !userID.Valid {
		return fail(fmt.Sprintf("usuário não resolvível (metadata=%q customer=%q)", info.UserID, info.CustomerID))
	}
	if !planID.Valid {
		return fail(fmt.Sprintf("plano não resolvível (price=%q metadata=%q)", info.PriceID, info.PlanID))
	}

	customerID := info.CustomerID
	if customerID == "" && hasExisting {
		customerID = existing.StripeCustomerID
	}
	if customerID == "" {
		return fail("customer id não resolvível")
	}

	// 3. Mantém o mapeamento cliente <-> usuário (fallback p/ eventos futuros).
	if err := store.UpsertStripeCustomer(ctx, sqlc.UpsertStripeCustomerParams{
		UserId:           userID,
		StripeCustomerId: customerID,
	}); err != nil {
		return "", err
	}

	// 4. Status / cancelamento.
	status := info.Status
	if status == "" && hasExisting {
		status = existing.Status
	}
	cancelAtPeriodEnd := info.CancelAtPeriodEnd
	canceledAt := timestampFromUnix(info.CanceledAt)

	if opts.forceCanceled {
		status = "canceled"
		cancelAtPeriodEnd = false
		if !canceledAt.Valid {
			canceledAt = pgtype.Timestamp{Time: time.Now().UTC(), Valid: true}
		}
	}
	if status == "" {
		status = "incomplete"
	}

	// 5. UPSERT — checkout_session_id é passado como NULL para preservar o
	//    valor já gravado pelo checkout (COALESCE na query).
	if _, err := store.UpsertSubscription(ctx, sqlc.UpsertSubscriptionParams{
		UserId:                  userID,
		PlanId:                  planID,
		StripeCustomerID:        customerID,
		StripeSubscriptionID:    info.ID,
		StripeCheckoutSessionID: pgtype.Text{},
		StripePriceID:           pgtype.Text{String: info.PriceID, Valid: info.PriceID != ""},
		Status:                  status,
		CurrentPeriodStart:      timestampFromUnix(info.CurrentPeriodStart),
		CurrentPeriodEnd:        timestampFromUnix(info.CurrentPeriodEnd),
		CancelAtPeriodEnd:       cancelAtPeriodEnd,
		CanceledAt:              canceledAt,
		UpdatedAt:               pgtype.Timestamp{Time: time.Now().UTC(), Valid: true},
	}); err != nil {
		return "", err
	}

	// 6. Regra de acesso na MESMA transação (spec seção 17).
	if err := syncSubscriptionAccess(ctx, store, userID, planID, status); err != nil {
		return "", err
	}

	return opts.message, nil
}

// syncSubscriptionAccess concede/revoga APENAS o papel do plano desta
// assinatura, conforme a política HasActiveAccess.
//
//   - cancel_at_period_end=true não remove acesso (status continua active).
//   - A revogação definitiva ocorre quando status vira "canceled"
//     (customer.subscription.deleted).
func syncSubscriptionAccess(ctx context.Context, store stripeWebhookStore, userID, planID pgtype.UUID, status string) error {
	roleID, err := store.GetPlanRoleId(ctx, planID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("plano %s sem linha/role; acesso não sincronizado", planID)
			return nil
		}
		return err
	}
	if !roleID.Valid {
		return nil
	}

	if services.HasActiveAccess(status) {
		// Idempotente (ON CONFLICT DO NOTHING).
		return store.AssignRoleToUser(ctx, sqlc.AssignRoleToUserParams{
			UserId: userID,
			RoleId: roleID,
		})
	}
	// Remove somente este UserRole específico (não toca em Admin nem em
	// roles de outras assinaturas/planos).
	return store.RemoveRoleFromUser(ctx, sqlc.RemoveRoleFromUserParams{
		UserId: userID,
		RoleId: roleID,
	})
}

// ---------------------------------------------------------------------------
// invoice.paid | invoice.payment_failed
// ---------------------------------------------------------------------------

// handleInvoiceEvent registra a cobrança na tabela StripeInvoices.
//
//   - invoice.paid          → status "paid" + PaidAt.
//   - invoice.payment_failed → status "payment_failed"; NÃO cancela a
//     assinatura nem altera o Status da Subscription (o Stripe reenviará
//     customer.subscription.updated com past_due, se aplicável).
//
// Nunca cria uma Subscription: o estado oficial vem dos eventos de
// customer.subscription.*.
func handleInvoiceEvent(ctx context.Context, store stripeWebhookStore, raw []byte, status string) (string, error) {
	info, err := extractInvoiceEventData(raw)
	if err != nil {
		return "", err
	}
	if info.ID == "" {
		return "", errors.New("evento de invoice sem id")
	}

	params := sqlc.UpsertStripeInvoiceParams{
		StripeInvoiceId:      info.ID,
		StripeSubscriptionId: pgtype.Text{String: info.SubscriptionID, Valid: info.SubscriptionID != ""},
		StripeCustomerId:     pgtype.Text{String: info.CustomerID, Valid: info.CustomerID != ""},
		AmountPaid:           pgtype.Int8{Int64: info.AmountPaid, Valid: info.AmountPaid > 0 || status == "paid"},
		Currency:             pgtype.Text{String: info.Currency, Valid: info.Currency != ""},
		Status:               status,
	}
	if status == "paid" {
		params.PaidAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	}

	// A assinatura pode ainda não existir no banco (ordem fora de sequência);
	// os campos são nullable justamente para isso.
	if err := store.UpsertStripeInvoice(ctx, params); err != nil {
		return "", err
	}

	if status == "paid" {
		return "Cobrança paga registrada.", nil
	}
	log.Printf("invoice.payment_failed %s (subscription=%s): falha registrada sem alterar a assinatura", info.ID, info.SubscriptionID)
	return "Falha de pagamento registrada.", nil
}

// ---------------------------------------------------------------------------
// product.* / price.* (sincronização de planos — comportamento preservado)
// ---------------------------------------------------------------------------

func handlePriceSyncEvent(ctx context.Context, store stripeWebhookStore, raw []byte) (string, error) {
	var price stripe.Price
	if err := json.Unmarshal(raw, &price); err != nil {
		return "", fmt.Errorf("parse price: %w", err)
	}

	var productID string
	if price.Product != nil {
		productID = price.Product.ID
	}
	if productID == "" {
		return "price without product, skipping", nil
	}

	return CreatePlan(ctx, store, Plan{
		StripeProductID: productID,
		StripePriceID:   price.ID,
		Price:           price.UnitAmount,
		Active:          price.Active,
		DeactivateOnly:  true,
	})
}

func handleProductSyncEvent(ctx context.Context, store stripeWebhookStore, raw []byte) (string, error) {
	var product stripe.Product
	if err := json.Unmarshal(raw, &product); err != nil {
		return "", fmt.Errorf("parse product: %w", err)
	}

	var stripePriceID string
	if product.DefaultPrice != nil {
		stripePriceID = product.DefaultPrice.ID
	}

	if !product.Active {
		if err := store.DeactivatePlan(ctx, sqlc.DeactivatePlanParams{
			StripeProductID: product.ID,
			UpdatedAt:       pgtype.Timestamp{Time: time.Now().UTC(), Valid: true},
		}); err != nil {
			return "", err
		}
	}

	return CreatePlan(ctx, store, Plan{
		Name:            product.Name,
		Description:     product.Description,
		StripeProductID: product.ID,
		StripePriceID:   stripePriceID,
		Active:          product.Active,
		RoleId:          DefaultPlanRoleID,
	})
}

func handleProductDeletedEvent(ctx context.Context, store stripeWebhookStore, raw []byte) (string, error) {
	var product stripe.Product
	if err := json.Unmarshal(raw, &product); err != nil {
		return "", fmt.Errorf("parse product: %w", err)
	}

	return DeactivatePlan(ctx, store, Plan{
		Name:            product.Name,
		Description:     product.Description,
		StripeProductID: product.ID,
		Active:          product.Active,
		RoleId:          DefaultPlanRoleID,
	})
}
