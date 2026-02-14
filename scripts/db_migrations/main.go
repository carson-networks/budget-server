package main

import (
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	server_config "github.com/carson-networks/budget-server/internal/config"
)

func main() {
	env, err := server_config.ProcessEnvironmentVariables()
	if err != nil {
		logrus.WithError(err).Fatal("ProcessEnvironmentVariables")
		return
	}

	connectionDetails := "postgres://" + env.PostgresUsername + ":" + env.PostgresPassword + "@" + env.PostgresAddress + ":" + env.PostgresPort + "/" + env.PostgresDB + "?sslmode=disable"
	println(connectionDetails)

	db, err := sql.Open("postgres", connectionDetails)
	if err != nil {
		logrus.WithError(err).Fatal("sql.Open")
		return
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logrus.WithError(err).Fatal("postgres.WithInstance")
		return
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		logrus.WithError(err).Fatal("migrate.NewWithDatabaseInstance")
		return
	}

	preMigrationVersion, _, err := m.Version()
	if err != nil && errors.Is(err, migrate.ErrNilVersion) {
		preMigrationVersion = 0
	} else if err != nil {
		logrus.WithError(err).Fatal("m.Version.preMigrationVersion")
		return
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logrus.WithError(err).Fatal()
		return
	}

	postMigrationVersion, _, err := m.Version()
	if err != nil {
		logrus.WithError(err).Fatal("m.Version.postMigrationVersion")
		return
	}

	logrus.WithFields(logrus.Fields{
		"preMigrationVersion":  preMigrationVersion,
		"postMigrationVersion": postMigrationVersion,
	}).Info("Migration status")

}
