package services

// HasActiveAccess define a política de acesso do produto para os status de
// assinatura reportados pelo Stripe.
//
// Política:
//   - "active"           → acesso liberado
//   - "trialing"         → acesso liberado (período de teste)
//   - "past_due"         → acesso liberado (período de carência enquanto o
//     Stripe faz novas tentativas de cobrança; o estado oficial continua
//     sendo sincronizado pelos eventos customer.subscription.*)
//   - "canceled"         → sem acesso
//   - "unpaid"           → sem acesso
//   - "incomplete"       → sem acesso (primeira cobrança ainda não confirmada)
//   - "incomplete_expired"| "paused" | qualquer outro → sem acesso
//
// Observações:
//   - cancel_at_period_end = true NÃO remove acesso: o status continua
//     "active" até que customer.subscription.deleted seja recebido.
//   - A revogação definitiva acontece apenas quando o Stripe envia
//     customer.subscription.deleted (status "canceled").
func HasActiveAccess(status string) bool {
	switch status {
	case "active", "trialing", "past_due":
		return true
	default:
		return false
	}
}
