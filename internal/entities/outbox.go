package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID            uuid.UUID
	EventType     string
	EventVersion  int
	AggregateType string
	AggregateID   uuid.UUID
	Payload       json.RawMessage
	CreatedAt     time.Time
	PublishedAt   *time.Time
	AvailableAt   time.Time
	LockedUntil   *time.Time
	Attempts      int
	LastError     *string
}
