package app

import (
	"errors"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	_defaultAttempts = 20
	_defaultTimeout  = time.Second
)

func Migrate(databaseURL string) error {
	var (
		attempts = _defaultAttempts
		err      error
		m        *migrate.Migrate
	)
	for attempts > 0 {
		m, err = migrate.New("file://migrations", databaseURL)
		if err == nil {
			break
		}
		log.Printf("Migrate: postgres is trying to connect, attempts left: %d", attempts)
		log.Printf("Migrate: error occurred while trying to connect to postgres: %v", err)
		time.Sleep(_defaultTimeout)
		attempts--
	}

	if err != nil {
		return errors.Join(errors.New("migrate: connection attempts exhausted"), err)
	}

	defer func() {
		_, _ = m.Close()
	}()

	err = m.Up()
	defer m.Close()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return errors.Join(errors.New("migrate: up"), err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}

	return nil
}
