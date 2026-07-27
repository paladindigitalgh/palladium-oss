// Package migrations embeds the SQL migration files so they ship inside the
// compiled binary and can be applied without access to the source tree. The
// .sql files here are also plain goose migrations and can be run directly
// with the goose CLI against this directory for ad hoc operational use.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
