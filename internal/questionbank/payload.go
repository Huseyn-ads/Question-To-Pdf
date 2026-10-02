package questionbank

import "time"

type CreateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Response struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpdateRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}
