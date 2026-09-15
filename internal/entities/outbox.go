package entities

import (
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	id uuid.UUID
	EventType string
	EventVersion string
	AggregateType string
	AggregateID uuid.UUID
	Payload []byte
	CreatedAt time.Time
	PublishedAt *time.Time
}