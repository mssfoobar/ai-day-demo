package repo

// TODELETE(scaffold): Delete this file and replace with real entity types

import (
	"time"

	"github.com/google/uuid"
	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
)

// ExampleRow is the DB-layer representation of an example. Real repos use
// `db` tags matching their SQL schema columns. This scaffold uses an
// in-memory sink so the service compiles and tests pass without a real DB.
type ExampleRow struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	ReportedBy  string    `db:"reported_by"`
	TenantID    string    `db:"tenant_id"`
	CreatedAt   time.Time `db:"created_at"`
}

// ExampleFilter carries the list query inputs (tenant scope + pagination + free-form filters).
// Real repos translate this into WHERE / LIMIT / OFFSET / ORDER BY clauses.
//
// `Page` is the parsed AOH-canonical pagination block (`page`/`size`/repeated
// `sort` query params) produced by `aohhttp.GetQueryPagination`. Real repos
// MUST whitelist `Page.Sorts[i].Column` against an allowlist before composing
// ORDER BY — never interpolate user-supplied column names directly into SQL.
type ExampleFilter struct {
	TenantID string
	Page     aohhttp.PageRequest
}
