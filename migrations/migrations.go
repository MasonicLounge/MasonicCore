package migrations

import "embed"

// FS embeds all *.sql migration files into the binary.
//
//go:embed *.sql
var FS embed.FS
