package bobgen

//go:generate go run github.com/stephenafamo/bob/gen/bobgen-psql@latest -c ../../../../../bobgen.yaml
//
// Prerequisites: PostgreSQL must be running with the transactions table created.
// 1. Run migrations: go run ./scripts/db_migrations/
// 2. Then run: go generate ./internal/storage/models/
// Override DSN: PSQL_DSN=postgres://user:pass@host:port/db?sslmode=disable
