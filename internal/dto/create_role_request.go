package dto

type CreateRoleRequest struct {
	Name string `json:"name" validate:"required,max=50"`
}