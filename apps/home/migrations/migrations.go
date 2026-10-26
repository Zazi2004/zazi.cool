// Package migrations embeds the SQL migration files in the binary.
package migrations

import "embed"

// FS holds the *.sql files of this directory, applied in lexical order.
//
//go:embed *.sql
var FS embed.FS
