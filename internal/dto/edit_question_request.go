package dto

type EditQuestion struct {
	Answer string   `json:"answer" validate:"required,min=2,max=2000"`
}
