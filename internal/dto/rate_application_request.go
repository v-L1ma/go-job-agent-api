package dto

type RateApplicationRequest struct {
    Liked           bool `json:"liked" validate:"required"`
    Feedback 		string `json:"feedback" validate:"omitempty,min=3,max=600"`
}