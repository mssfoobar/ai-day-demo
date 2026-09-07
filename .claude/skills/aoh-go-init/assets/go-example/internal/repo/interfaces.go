package repo

// TODELETE(scaffold): Delete this file and replace with real entity interfaces

import "context"

type ExampleRepository interface {
	Create(ctx context.Context, row ExampleRow) (ExampleRow, error)
	List(ctx context.Context, filter ExampleFilter) (rows []ExampleRow, total int64, err error)
}
