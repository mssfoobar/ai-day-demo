package aohhttp

import (
	"net/http"
	"time"

	"github.com/go-chi/render"
)

type ResponsePayload struct {
	HttpStatusCode int    `json:"-"`
	Data           any    `json:"data,omitempty"`
	Message        string `json:"message,omitempty"`
	SentAt         string `json:"sent_at,omitempty"`
}

func (res *ResponsePayload) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, res.HttpStatusCode)
	return nil
}

func Response(code int, message string, data any) render.Renderer {
	return &ResponsePayload{
		HttpStatusCode: code,
		Data:           data,
		Message:        message,
		SentAt:         time.Now().UTC().Format(time.RFC3339),
	}
}

type PaginationResponsePayload struct {
	ResponsePayload
	Page PageResponse `json:"page,omitempty"`
}

func PaginationResponse(code int, message string, page PageResponse, data any) render.Renderer {
	return &PaginationResponsePayload{
		ResponsePayload: ResponsePayload{
			HttpStatusCode: code,
			Data:           data,
			Message:        message,
			SentAt:         time.Now().UTC().Format(time.RFC3339),
		},
		Page: page,
	}
}
