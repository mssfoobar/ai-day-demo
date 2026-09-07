package repo

// TODELETE(scaffold): Delete this file — example entity for reference only.
// A real repo runs SQL via `r.db.conn`. This one is an in-memory stub so the
// scaffold compiles and `make test` passes without a live Postgres.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type exampleRepository struct {
	db *DB
}

func NewExampleRepository(db *DB) ExampleRepository {
	return &exampleRepository{db: db}
}

func (r *exampleRepository) Create(ctx context.Context, row ExampleRow) (ExampleRow, error) {
	// Real pattern (replace with this when wiring your entity):
	//
	//   const insertSQL = `INSERT INTO examples (id, name, description,
	//       reported_by, tenant_id) VALUES ($1, $2, $3, $4, $5)
	//       RETURNING id, name, description, reported_by, tenant_id, created_at`
	//
	//   id, err := uuid.NewV7()
	//   if err != nil {
	//       return ExampleRow{}, fmt.Errorf("generate id: %w", err)
	//   }
	//   var out ExampleRow
	//   if err := r.db.conn.GetContext(ctx, &out, insertSQL,
	//       id, row.Name, row.Description, row.ReportedBy, row.TenantID,
	//   ); err != nil {
	//       return ExampleRow{}, fmt.Errorf("insert example: %w", err)
	//   }
	//   return out, nil

	id, err := uuid.NewV7()
	if err != nil {
		return ExampleRow{}, fmt.Errorf("generate id: %w", err)
	}
	row.ID = id
	row.CreatedAt = time.Now().UTC()
	return row, nil
}

func (r *exampleRepository) List(ctx context.Context, filter ExampleFilter) ([]ExampleRow, int64, error) {
	// Real pattern (replace with this when wiring your entity):
	//
	//   const countSQL = `SELECT COUNT(*) FROM examples WHERE tenant_id = $1`
	//   const listSQL  = `SELECT id, name, description, reported_by, tenant_id, created_at
	//                     FROM examples WHERE tenant_id = $1
	//                     ORDER BY %s LIMIT $2 OFFSET $3`
	//
	//   var total int64
	//   if err := r.db.conn.GetContext(ctx, &total, countSQL, filter.TenantID); err != nil {
	//       return nil, 0, fmt.Errorf("count examples: %w", err)
	//   }
	//
	//   // Build ORDER BY from filter.Page.Sorts; whitelist columns first.
	//   orderBy := buildOrderBy(filter.Page.Sorts) // e.g. "created_at DESC, id ASC"
	//   offset := (filter.Page.Number - 1) * filter.Page.Size
	//   var rows []ExampleRow
	//   if err := r.db.conn.SelectContext(ctx, &rows, fmt.Sprintf(listSQL, orderBy),
	//       filter.TenantID, filter.Page.Size, offset,
	//   ); err != nil {
	//       return nil, 0, fmt.Errorf("list examples: %w", err)
	//   }
	//   return rows, total, nil

	// In-memory stub: return empty slice so the scaffold compiles and tests pass.
	return []ExampleRow{}, 0, nil
}
