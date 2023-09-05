package api

type CollectionResult[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}
