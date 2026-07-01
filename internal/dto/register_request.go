package dto

type RegisterRequest struct {
	Name                string `json:"name" validate:"required,max=60"`
	Email               string `json:"email" validate:"required,email,max=50"`
	Password      		string `json:"password" validate:"required,min=6,max=50"`
	ConfirmPassword     string `json:"confirmPassword" validate:"required,min=6"`
}