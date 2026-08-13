package migrations

import "embed"

//go:embed core/*.sql
var CoreMigrationsFS embed.FS

//go:embed order/*.sql
var OrderMigrationsFS embed.FS
