// Package migrations exposes the embedded SQL migration files so the binary
// can apply them without a mounted volume.
package migrations

import "embed"

// FS holds every *.sql file under migrations/, embedded at build time.
//
//go:embed *.sql
var FS embed.FS
