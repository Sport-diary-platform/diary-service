package postgres

import (
	"errors"
	"testing"

	"diary-service/internal/entities"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapNotFoundPreservesDomainError(t *testing.T) {
	err := mapNotFound("test", pgx.ErrNoRows, entities.ErrWorkoutNotFound)
	if !errors.Is(err, entities.ErrWorkoutNotFound) {
		t.Fatalf("expected workout not found, got %v", err)
	}
}

func TestMapUniquePreservesDomainError(t *testing.T) {
	err := mapUnique("test", &pgconn.PgError{Code: "23505"}, entities.ErrDailyCheckInAlreadyExists)
	if !errors.Is(err, entities.ErrDailyCheckInAlreadyExists) {
		t.Fatalf("expected daily check-in already exists, got %v", err)
	}
}

func TestLocalDateValueRejectsInvalidDate(t *testing.T) {
	if _, err := localDateValue("2026-02-30"); err == nil {
		t.Fatal("expected invalid date error")
	}
}
