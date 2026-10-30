// Package migrations embeds the SQL migration files in the binary.
package migrations

import "embed"

// FS holds the *.sql files of this directory, applied in lexical order. The pattern is
// "*" rather than "*.sql" because go:embed fails when a pattern matches nothing, and the
// first migration (sessions) only arrives with the authentication ticket; migrate.Up
// ignores everything that is not a .sql file (here, this Go file).
//
//go:embed *
var FS embed.FS
