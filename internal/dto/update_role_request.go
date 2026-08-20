package dto

type UpdateRoleRequest struct {
	Name   string `json:"name" validate:"required,max=50"`
	Active bool   `json:"active"`
}