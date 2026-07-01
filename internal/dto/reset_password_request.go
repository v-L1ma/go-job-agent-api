package dto

type ResetPasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=6,max=50"`
	ConfirmPassword string `json:"confirmPassword" validate:"required,min=6"`
}
