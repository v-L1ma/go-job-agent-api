package dto

type ApplyRequest struct {
    JobId           string `json:"jobId"`
    Status 			string `json:"status"`
    Observation     string `json:"observation"`
	Platform 		string `json:"platform"`
}