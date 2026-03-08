package bobgen

//go:generate sh -c "cd ../../../../ && go run github.com/stephenafamo/bob/gen/bobgen-psql@latest -c bobgen.yaml"
//
// Run from repo root so destination "internal/storage/sqlconfig/bobgen" resolves to one output dir (no duplicate nested path).
// Prerequisites: PostgreSQL running with migrations applied.
// 1. Run migrations: make migrate (or go run ./scripts/db_migrations from that dir)
// 2. Run: go generate ./internal/storage/sqlconfig/bobgen/
// Override DSN: PSQL_DSN=postgres://user:pass@host:port/db?sslmode=disable
