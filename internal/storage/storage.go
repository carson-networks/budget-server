package storage

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"

	"github.com/carson-networks/budget-server/internal/config"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig"
)

type Storage struct {
	DB           *sql.DB
	Transactions sqlconfig.ITransactionTable
	Accounts     sqlconfig.IAccountTable
}

func NewStorage(env *config.Config) *Storage {
	connStr := "postgres://" + env.PostgresUsername + ":" +
		env.PostgresPassword + "@" + env.PostgresAddress + ":" +
		env.PostgresPort + "/" + env.PostgresDB + "?sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	transactionsTable := sqlconfig.NewTransactionsTable(db)
	accountsTable := sqlconfig.NewAccountsTable(db)
	return &Storage{
		DB:           db,
		Transactions: &transactionsTable,
		Accounts:     &accountsTable,
	}
}
