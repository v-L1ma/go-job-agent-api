package services

import "os"

// StripeSecretKey retorna a chave secreta usada para falar com a API do
// Stripe. O nome principal é STRIPE_SECRET_KEY (spec); STRIPE_API_KEY é
// mantido como fallback para compatibilidade com configurações antigas.
func StripeSecretKey() string {
	if v := os.Getenv("STRIPE_SECRET_KEY"); v != "" {
		return v
	}
	return os.Getenv("STRIPE_API_KEY")
}

// StripeWebhookSecret retorna o secret usado para validar o header
// Stripe-Signature dos webhooks.
func StripeWebhookSecret() string {
	return os.Getenv("STRIPE_WEBHOOK_SECRET")
}
