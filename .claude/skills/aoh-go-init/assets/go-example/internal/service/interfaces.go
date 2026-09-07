package service

// TODELETE(scaffold): Delete this file and replace with real entity interfaces

import (
	"context"
	"time"

	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
)

const (
	// ExampleNameMaxLen / ExampleDescriptionMaxLen mirror the column CHECK
	// constraints you'd put on the SQL schema. Keep the Go constants and the
	// DB CHECK in sync when you adjust them.
	ExampleNameMaxLen        = 100
	ExampleDescriptionMaxLen = 1000
)

// Example is the HTTP response / domain shape.
type Example struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ReportedBy  string    `json:"reported_by"`
	TenantID    string    `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateExampleInput is the service-layer input. Handlers assemble this from
// the request body + context-extracted JWT claims and pass it in.
type CreateExampleInput struct {
	Name        string
	Description string
	ReportedBy  string
	TenantID    string
}

// ListExamplesInput carries the list query: tenant scope from JWT + pagination
// (and any free-form filters from query params, added per-entity).
//
// `Page` is the parsed AOH-canonical pagination block (page number, size,
// repeated sort clauses) produced by `aohhttp.GetQueryPagination`. Empty
// `Page.Sorts` means "use the default" — the service applies `created_at,desc`.
type ListExamplesInput struct {
	TenantID string
	Page     aohhttp.PageRequest
}

type ExampleService interface {
	Create(ctx context.Context, in CreateExampleInput) (Example, error)
	List(ctx context.Context, in ListExamplesInput) (results []Example, total int64, err error)
}
