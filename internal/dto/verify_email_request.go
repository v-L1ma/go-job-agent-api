package dto

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}
