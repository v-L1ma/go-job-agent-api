package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/resend/resend-go/v4"
)

// Emailer é a abstração mínima de envio de e-mail transacional via Resend.
type Emailer interface {
	// Send envia um e-mail com corpo HTML puro (sem template).
	Send(ctx context.Context, to, subject, htmlBody string) error
	// SendWithTemplate envia um e-mail usando um template Resend existente.
	// O nome do template (ex.: "password-reset", "account-confirmation") vem
	// do parâmetro, assim como as variáveis que o template espera.
	SendWithTemplate(ctx context.Context, to, templateName string, variables map[string]interface{}) error
}

// ResendEmailer envia e-mails transacionais usando o Resend.
type ResendEmailer struct {
	apiKey string
	from   string
}

// NewEmailer cria o emailer a partir das variáveis de ambiente.
//
// Requisitos:
//   - RESEND_API_KEY: chave da API Resend (obrigatória em produção).
//   - EMAIL_FROM: remetente opcional; se vazio, usa "Acme <onboarding@resend.dev>".
func NewEmailer() Emailer {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		log.Fatal("RESEND_API_KEY environment variable is required")
	}

	from := os.Getenv("EMAIL_FROM")
	if from == "" {
		from = "Acme <onboarding@resend.dev>"
	}

	return &ResendEmailer{
		apiKey: apiKey,
		from:   from,
	}
}

// Send entrega o e-mail via Resend API (corpo HTML puro).
func (e *ResendEmailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	client := resend.NewClient(e.apiKey)

	params := &resend.SendEmailRequest{
		From:    e.from,
		To:      []string{to},
		Subject: subject,
		Html:    htmlBody,
		Text:    stripHTML(htmlBody),
	}

	result, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("resend: error sending email: %w", err)
	}

	log.Printf("[resend] email sent successfully! ID: %s", result.Id)
	return nil
}

// SendWithTemplate envia um e-mail usando um template Resend. O `From` é o
// mesmo usado pelo fluxo de redefinição de senha ("Contato
// <contato@trampo.click>") para manter o comportamento de envio existente.
func (e *ResendEmailer) SendWithTemplate(ctx context.Context, to, templateName string, variables map[string]interface{}) error {
	client := resend.NewClient(e.apiKey)

	params := &resend.SendEmailRequest{
		From: "Contato <contato@trampo.click>",
		To:   []string{to},
		Template: &resend.EmailTemplate{
			Id:        templateName,
			Variables: variables,
		},
	}

	result, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("resend: error sending %s template email: %w", templateName, err)
	}

	log.Printf("[resend] %s template email sent successfully! ID: %s", templateName, result.Id)
	return nil
}

// FrontendURL retorna a base URL do frontend usada para montar links:
func FrontendURL() string {
	v := strings.TrimRight(strings.TrimSpace(os.Getenv("FRONTEND_URL")), "/")
	if v == "" {
		return "http://localhost:3000"
	}
	return v
}

// stripHTML removes basic HTML tags to produce a plain-text fallback.
func stripHTML(s string) string {
	const openTag = "<"
	const closeTag = ">"
	var result strings.Builder
	inTag := false
	for _, ch := range s {
		if ch == '<' {
			inTag = true
		} else if ch == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(ch)
		}
	}
	return result.String()
}