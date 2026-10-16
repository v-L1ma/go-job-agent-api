package services

// Enforcement dos limites diários por plano de assinatura.
//
// Política de "sem limite" (mantém o comportamento anterior para quem não
// tem assinatura paga):
//   - usuário sem registro em "Subscriptions"            → sem limite;
//   - assinatura com status sem acesso (HasActiveAccess
//     falso: canceled, unpaid, incomplete, etc.)         → sem limite;
//   - plano com o limite correspondente NULL             → sem limite;
//   - assinatura cujo plano não existe mais (defensivo)  → sem limite.
//
// Ou seja, o limite só é aplicado a usuários com assinatura ativa
// ("active", "trialing" ou "past_due") cujo plano define um limite.
// O contador é diário em UTC (meia-noite UTC como início da janela).

import (
	"context"
	"errors"
	"fmt"
	"time"

	sqlc "job-agent-api/internal/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// PlanLimitStore agrupa as queries sqlc usadas no enforcement de limites.
// *sqlc.Queries (db.Query) satisfaz a interface.
type PlanLimitStore interface {
	GetSubscriptionByUserId(ctx context.Context, userID pgtype.UUID) (sqlc.Subscription, error)
	GetPlanById(ctx context.Context, id pgtype.UUID) (sqlc.GetPlanByIdRow, error)
	CountUserGeneratedCvsSince(ctx context.Context, arg sqlc.CountUserGeneratedCvsSinceParams) (int32, error)
	CountUserApplicationsSince(ctx context.Context, arg sqlc.CountUserApplicationsSinceParams) (int32, error)
	CountUserCvAnalysesSince(ctx context.Context, arg sqlc.CountUserCvAnalysesSinceParams) (int32, error)
}

// ErrPlanLimitExceeded é o sentinel para "usuário atingiu o limite diário do
// plano". Os handlers devem mapeá-lo para HTTP 429 com a mensagem completa
// de erro (err.Error()) — que já inclui o valor do limite.
var ErrPlanLimitExceeded = errors.New("limite diário do plano atingido")

// PlanLimitError retorna quando o uso do dia já atingiu o limite do plano.
type PlanLimitError struct {
	Limit    int32
	Resource string // "currículos" | "candidaturas" | "análises de currículo"
}

func (e *PlanLimitError) Error() string {
	return fmt.Sprintf("Você atingiu o limite diário de %d %s do seu plano. Tente novamente amanhã.", e.Limit, e.Resource)
}

func (e *PlanLimitError) Unwrap() error { return ErrPlanLimitExceeded }

// StartOfUtcDay retorna o início do dia corrente em UTC (meia-noite UTC),
// usado como janela "desde" nas contagens diárias.
func StartOfUtcDay() pgtype.Timestamptz {
	// time.Truncate arredonda para baixo múltiplos de d desde o instante zero
	// (01/01/0001 00:00:00 UTC); com 24h o resultado é sempre meia-noite UTC.
	now := time.Now().UTC()
	return pgtype.Timestamptz{
		Time:  time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
}

// planOfActiveSubscription resolve o plano da assinatura ativa do usuário.
// ok=false significa "sem limite" (sem assinatura, sem acesso ou plano ausente).
func planOfActiveSubscription(ctx context.Context, store PlanLimitStore, userID pgtype.UUID) (sqlc.GetPlanByIdRow, bool, error) {
	sub, err := store.GetSubscriptionByUserId(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.GetPlanByIdRow{}, false, nil
	}
	if err != nil {
		return sqlc.GetPlanByIdRow{}, false, err
	}

	if !HasActiveAccess(sub.Status) {
		return sqlc.GetPlanByIdRow{}, false, nil
	}

	plan, err := store.GetPlanById(ctx, sub.PlanId)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.GetPlanByIdRow{}, false, nil
	}
	if err != nil {
		return sqlc.GetPlanByIdRow{}, false, err
	}

	return plan, true, nil
}

// CheckDailyCvLimit bloqueia (PlanLimitError / ErrPlanLimitExceeded) quando o
// usuário já gerou "DailyCvLimit" currículos hoje (UTC). Limite 0 bloqueia
// sempre; NULL deixa ilimitado. Chamado antes da geração por IA para evitar
// o custo do Gemini em chamadas que seriam rejeitadas.
func CheckDailyCvLimit(ctx context.Context, store PlanLimitStore, userID pgtype.UUID) error {
	plan, ok, err := planOfActiveSubscription(ctx, store, userID)
	if err != nil {
		return err
	}
	if !ok || !plan.DailyCvLimit.Valid {
		return nil
	}

	used, err := store.CountUserGeneratedCvsSince(ctx, sqlc.CountUserGeneratedCvsSinceParams{
		UserId:    userID,
		CreatedAt: StartOfUtcDay(),
	})
	if err != nil {
		return err
	}

	if used >= plan.DailyCvLimit.Int32 {
		return &PlanLimitError{Limit: plan.DailyCvLimit.Int32, Resource: "currículos"}
	}
	return nil
}

// CheckDailyApplicationLimit bloqueia (PlanLimitError / ErrPlanLimitExceeded)
// quando o usuário já fez "DailyApplicationLimit" candidaturas hoje (UTC).
func CheckDailyApplicationLimit(ctx context.Context, store PlanLimitStore, userID pgtype.UUID) error {
	plan, ok, err := planOfActiveSubscription(ctx, store, userID)
	if err != nil {
		return err
	}
	if !ok || !plan.DailyApplicationLimit.Valid {
		return nil
	}

	used, err := store.CountUserApplicationsSince(ctx, sqlc.CountUserApplicationsSinceParams{
		UserId:    userID,
		CreatedAt: StartOfUtcDay(),
	})
	if err != nil {
		return err
	}

	if used >= plan.DailyApplicationLimit.Int32 {
		return &PlanLimitError{Limit: plan.DailyApplicationLimit.Int32, Resource: "candidaturas"}
	}
	return nil
}

// CheckDailyCvAnalysisLimit bloqueia (PlanLimitError / ErrPlanLimitExceeded)
// quando o usuário já rodou "DailyCvLimit" análises de currículo hoje (UTC).
//
// Reutiliza o mesmo campo de limite do plano (Plans."DailyCvLimit") usado
// para currículos gerados, porém com um contador próprio: a tabela
// contada é "CvAnalyses" (análises de score/melhorias), não
// "GeneratedCvsNew". Chamado antes da chamada à IA para evitar o custo do
// Gemini em requisições que serão rejeitadas.
func CheckDailyCvAnalysisLimit(ctx context.Context, store PlanLimitStore, userID pgtype.UUID) error {
	plan, ok, err := planOfActiveSubscription(ctx, store, userID)
	if err != nil {
		return err
	}
	if !ok || !plan.DailyCvLimit.Valid {
		return nil
	}

	used, err := store.CountUserCvAnalysesSince(ctx, sqlc.CountUserCvAnalysesSinceParams{
		UserId:    userID,
		CreatedAt: StartOfUtcDay(),
	})
	if err != nil {
		return err
	}

	if used >= plan.DailyCvLimit.Int32 {
		return &PlanLimitError{Limit: plan.DailyCvLimit.Int32, Resource: "análises de currículo"}
	}
	return nil
}
