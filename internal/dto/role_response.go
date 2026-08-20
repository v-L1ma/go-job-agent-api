package dto

type RoleResponse struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Active         bool   `json:"active"`
	CreatedBy      string `json:"createdBy"`
	CreatedAt      string `json:"createdAt"`
	LastModifiedBy string `json:"lastModifiedBy"`
	LastModifiedAt string `json:"lastModifiedAt"`
}