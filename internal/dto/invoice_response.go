package dto

import (
	"time"

	sqlc "job-agent-api/internal/queries"

	"github.com/jackc/pgx/v5/pgtype"
)

// InvoiceResponse representa uma fatura do Stripe (tabela "StripeInvoices")
// no painel admin (camelCase).
//
// Nullability (contrato):
//   - amountPaid: emitido EM REAIS (float). O banco guarda centavos como
//     BIGINT NULL; quando NULL é emitido como 0 (nunca null).
//   - currency: "" quando NULL no banco.
//   - paidAt: null quando NULL no banco.
//   - createdAt: TIMESTAMPTZ NOT NULL → sempre presente (RFC3339).
type InvoiceResponse struct {
	Id         string  `json:"id"`
	AmountPaid float64 `json:"amountPaid"`
	Currency   string  `json:"currency"`
	Status     string  `json:"status"`
	PaidAt     *string `json:"paidAt"`
	CreatedAt  string  `json:"createdAt"`
}

// MapInvoiceResponse converte a linha de GetInvoicesByStripeSubscriptionId
// para o DTO admin, com o valor convertido de centavos (BIGINT) para reais.
func MapInvoiceResponse(inv sqlc.GetInvoicesByStripeSubscriptionIdRow) InvoiceResponse {
	response := InvoiceResponse{
		Id:         inv.StripeInvoiceId,
		AmountPaid: 0, // AmountPaid NULL no banco → 0 (nunca null)
		Currency:   inv.Currency.String,
		Status:     inv.Status,
		CreatedAt:  timestamptzToRFC3339(inv.CreatedAt),
	}

	if inv.AmountPaid.Valid {
		response.AmountPaid = CentsToBRLInt64(inv.AmountPaid.Int64)
	}
	if inv.PaidAt.Valid {
		paidAt := inv.PaidAt.Time.Format(time.RFC3339)
		response.PaidAt = &paidAt
	}

	return response
}

// timestamptzToRFC3339 converte um pgtype.Timestamptz para string
// RFC3339. Retorna string vazia quando inválido (NULL no banco) — para
// colunas NOT NULL isso nunca acontece; é apenas defensivo.
func timestamptzToRFC3339(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.Format(time.RFC3339)
}
