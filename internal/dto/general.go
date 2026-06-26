package dto

type ResponseBase[T any] struct {
	Message string `json:"message,omitempty"`
	Data T `json:"data,omitempty"`
}