package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
)

type ProfileType string

const (
	ProfileTypeAthlete ProfileType = "athlete"
	ProfileTypeCoach   ProfileType = "coach"
)

// ProfileProvider supplies the business profile type for an authenticated user.
// Implementations normally call profile-service with a bounded timeout.
type ProfileProvider interface {
	GetProfileType(ctx context.Context, userID uuid.UUID) (ProfileType, error)
}

var (
	ErrForbidden                 = entities.ErrForbidden
	ErrInvalidCommand            = entities.ErrInvalidInput
	ErrInvalidProfileType        = entities.ErrInvalidProfileType
	ErrRelationshipAlreadyExists = entities.ErrRelationshipAlreadyExists
	ErrCommentNotFound           = entities.ErrCommentNotFound
)

func operationError(operation string, err error) error {
	if err == nil {
		return nil
	}

	return errors.Join(errors.New("usecase - "+operation), err)
}

func invalid(field, reason string) error {
	return errors.Join(ErrInvalidCommand, fmt.Errorf("%s: %s", field, reason))
}

func requireActor(actorID uuid.UUID) error {
	if actorID == uuid.Nil {
		return invalid("actor_id", "must be set")
	}

	return nil
}

func requireProfile(ctx context.Context, profiles ProfileProvider, actorID uuid.UUID, expected ProfileType) error {
	if err := requireActor(actorID); err != nil {
		return err
	}

	profileType, err := profiles.GetProfileType(ctx, actorID)
	if err != nil {
		return operationError("ProfileProvider.GetProfileType", err)
	}
	if profileType != ProfileTypeAthlete && profileType != ProfileTypeCoach {
		return errors.Join(ErrInvalidProfileType, fmt.Errorf("profile type %q", profileType))
	}
	if profileType != expected {
		return ErrForbidden
	}

	return nil
}

func requireActiveRelationship(ctx context.Context, relationships repository.RelationshipRepository, coachID, athleteID uuid.UUID) error {
	active, err := relationships.IsActive(ctx, coachID, athleteID)
	if err != nil {
		return operationError("RelationshipRepository.IsActive", err)
	}
	if !active {
		return entities.ErrRelationshipNotActive
	}

	return nil
}

func runTransaction(ctx context.Context, tx repository.TransactionManager, operation string, fn func(context.Context) error) error {
	if err := tx.WithTransaction(ctx, fn); err != nil {
		return operationError(operation+" - TransactionManager.WithTransaction", err)
	}

	return nil
}

func createEvent(ctx context.Context, outbox repository.OutboxRepository, eventType, aggregateType string, aggregateID uuid.UUID, payload any, now time.Time) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return operationError("OutboxEvent.MarshalPayload", err)
	}

	event := &entities.OutboxEvent{
		ID:            uuid.New(),
		EventType:     eventType,
		EventVersion:  1,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Payload:       data,
		CreatedAt:     now,
		AvailableAt:   now,
	}
	if err := outbox.Create(ctx, event); err != nil {
		return operationError("OutboxRepository.Create", err)
	}

	return nil
}

func validateRequiredString(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return invalid(field, "must not be empty")
	}

	return nil
}

func validateJSON(field string, value json.RawMessage) error {
	if len(value) == 0 || !json.Valid(value) || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return invalid(field, "must be valid non-null JSON")
	}

	return nil
}

func validateLocalDate(field string, value entities.LocalDate) error {
	parsed, err := time.Parse(time.DateOnly, string(value))
	if err != nil || parsed.Format(time.DateOnly) != string(value) {
		return invalid(field, "must use YYYY-MM-DD format")
	}

	return nil
}

func validateDateRange(from, to entities.LocalDate) error {
	if err := validateLocalDate("from", from); err != nil {
		return err
	}
	if err := validateLocalDate("to", to); err != nil {
		return err
	}
	if string(from) > string(to) {
		return invalid("date_range", "from must not be after to")
	}

	return nil
}

func validExpectedVersion(version int64) error {
	if version <= 0 {
		return invalid("expected_version", "must be greater than zero")
	}

	return nil
}

func versionMatches(actual, expected int64) error {
	if err := validExpectedVersion(expected); err != nil {
		return err
	}
	if actual != expected {
		return entities.ErrVersionConflict
	}

	return nil
}

func validateTimeRange(from, to time.Time) error {
	if from.IsZero() || to.IsZero() {
		return invalid("time_range", "from and to must be set")
	}
	if from.After(to) {
		return invalid("time_range", "from must not be after to")
	}

	return nil
}
