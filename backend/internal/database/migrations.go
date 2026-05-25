package database

import "embed"

// Migrations holds the embedded SQL migration files bundled into the binary.
// Migration files live in the migrations/ sub-directory next to this file.
//
//go:embed migrations/*.sql
var Migrations embed.FS
