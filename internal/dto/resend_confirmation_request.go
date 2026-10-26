package dto

type ResendConfirmationRequest struct {
	Email string `json:"email" validate:"required,email"`
}
