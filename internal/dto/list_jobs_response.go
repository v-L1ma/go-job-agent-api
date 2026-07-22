package dto

type ListJobsResponse struct {
    Jobs       []Job `json:"jobs"`
    NextCursor string       `json:"nextCursor,omitempty"`
}

type Job struct {
	Id             string             `json:"id"`
	PlataformJobId string             `json:"plataformJobId"`
	Title          string             `json:"title"`
	Description    string             `json:"description"`
	Url            string             `json:"url"`
	IsApplied      bool               `json:"isApplied"`
	Status         string             `json:"status"`
	Active         bool               `json:"active"`
	CreatedBy      string             `json:"createdBy"`
	CreatedAt      string  			  `json:"createdAt"`
	LastModifiedBy string             `json:"lastModifiedBy"`
	LastModifiedAt string			  `json:"lastModifiedAt"`
	Platform       string             `json:"platform"`
	Company        string             `json:"company"`
	Score 		   float64          `json:"score"`
}