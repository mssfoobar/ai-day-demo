// Package migrations embeds the SQL migration files.
//
// It lives beside the .sql files because go:embed cannot reach outside its own package
// directory — this keeps `migrations/` at the service root, where it is conventional to
// look for it, rather than burying the SQL under internal/.
package migrations

import "embed"

//go:embed *.up.sql
var FS embed.FS
