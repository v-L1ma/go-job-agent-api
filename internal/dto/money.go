package dto

// CentsToBRL converte um valor de plano armazenado no banco (inteiro em
// centavos, vindo de stripe.Price.UnitAmount — ex.: 1500) para reais em
// float64 (ex.: 15.0).
//
// Deve ser usada SOMENTE nos endpoints de LEITURA (GET), para que as
// respostas da API emitam o preço em reais. Os caminhos de ESCRITA
// (webhooks CreatePlan/MergePlan, UpdatePlan e as queries sqlc) continuam
// trabalhando com centavos (int) — não converter lá.
func CentsToBRL(cents int32) float64 {
	return float64(cents) / 100
}

// CentsToBRLInt64 idem a CentsToBRL, mas para valores guardados como BIGINT
// (int64) — ex.: a coluna "AmountPaid" de "StripeInvoices", que o Stripe
// envia em centavos como inteiro de 64 bits.
//
// Assim como CentsToBRL, usar SOMENTE em endpoints de LEITURA (GET).
func CentsToBRLInt64(cents int64) float64 {
	return float64(cents) / 100
}
