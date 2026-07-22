package migrations

import "embed"

//go:embed *.sql
var CoreMigrationsFS embed.FS
var OrderMigrationsFS embed.FS
