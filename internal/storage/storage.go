package storage

import (
	"context"
	"log"

	_ "github.com/lib/pq"
	"github.com/stephenafamo/bob"

	"github.com/carson-networks/budget-server/internal/config"
)

type Storage struct {
	sql bob.DB
}

func (s *Storage) Read() *Reader {
	return NewReader(s.sql)
}

func (s *Storage) Write(ctx context.Context) (*Writer, error) {
	tx, err := s.sql.Begin(ctx)
	if err != nil {
		return nil, err
	}

	w := NewWriter(tx)
	return &w, nil
}

func NewStorage(env *config.Config) *Storage {
	connStr := "postgres://" + env.PostgresUsername + ":" +
		env.PostgresPassword + "@" + env.PostgresAddress + ":" +
		env.PostgresPort + "/" + env.PostgresDB + "?sslmode=disable"

	db, err := bob.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	return &Storage{sql: db}
}
