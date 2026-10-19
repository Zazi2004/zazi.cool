// Package migrations embeds the SQL migration files in the binary.
package migrations

import "embed"

// FS holds the *.sql files of this directory, applied in lexical order.
// The pattern is "*" rather than "*.sql" because go:embed fails when nothing matches,
// and the first migration only arrives with the paste feature.
//
//go:embed *
var FS embed.FS
