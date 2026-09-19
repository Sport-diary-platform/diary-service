package postgres

import (
	"context"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type outboxRepository struct{ db DB }

func NewOutboxRepository(db DB) repository.OutboxRepository { return &outboxRepository{db: db} }

func (r *outboxRepository) Create(ctx context.Context, event *entities.OutboxEvent) error {
	const op = "OutboxRepository.Create"
	_, err := dbFromContext(ctx, r.db).Exec(ctx, `
		INSERT INTO outbox_events
			(id, event_type, event_version, aggregate_type, aggregate_id, payload, created_at,
			 published_at, available_at, locked_until, attempts, last_error)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, event.ID, event.EventType,
		event.EventVersion, event.AggregateType, event.AggregateID, event.Payload, event.CreatedAt,
		event.PublishedAt, event.AvailableAt, event.LockedUntil, event.Attempts, event.LastError)
	return wrap(op, err)
}

const outboxColumns = `id, event_type, event_version, aggregate_type, aggregate_id, payload,
	created_at, published_at, available_at, locked_until, attempts, last_error`

func scanOutboxEvent(row pgx.Row) (*entities.OutboxEvent, error) {
	event := new(entities.OutboxEvent)
	err := row.Scan(&event.ID, &event.EventType, &event.EventVersion, &event.AggregateType,
		&event.AggregateID, &event.Payload, &event.CreatedAt, &event.PublishedAt,
		&event.AvailableAt, &event.LockedUntil, &event.Attempts, &event.LastError)
	return event, err
}

func (r *outboxRepository) ClaimUnpublished(ctx context.Context, limit int, lockedUntil time.Time) ([]entities.OutboxEvent, error) {
	const op = "OutboxRepository.ClaimUnpublished"
	rows, err := dbFromContext(ctx, r.db).Query(ctx, `
		WITH candidates AS (
			SELECT id
			FROM outbox_events
			WHERE published_at IS NULL
				AND available_at <= NOW()
				AND (locked_until IS NULL OR locked_until <= NOW())
			ORDER BY available_at, created_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE outbox_events AS events
		SET locked_until = $2
		FROM candidates
		WHERE events.id = candidates.id
		RETURNING events.`+outboxColumns, limit, lockedUntil)
	if err != nil {
		return nil, wrap(op, err)
	}
	defer rows.Close()

	events := make([]entities.OutboxEvent, 0)
	for rows.Next() {
		event, scanErr := scanOutboxEvent(rows)
		if scanErr != nil {
			return nil, wrap(op, scanErr)
		}
		events = append(events, *event)
	}
	if err = rows.Err(); err != nil {
		return nil, wrap(op, err)
	}
	return events, nil
}

func (r *outboxRepository) MarkPublished(ctx context.Context, eventID uuid.UUID, publishedAt time.Time) error {
	const op = "OutboxRepository.MarkPublished"
	tag, err := dbFromContext(ctx, r.db).Exec(ctx, `
		UPDATE outbox_events
		SET published_at=$2, locked_until=NULL, last_error=NULL
		WHERE id=$1 AND published_at IS NULL`, eventID, publishedAt)
	if err != nil {
		return wrap(op, err)
	}
	if tag.RowsAffected() == 0 {
		return wrap(op, pgx.ErrNoRows)
	}
	return nil
}

func (r *outboxRepository) MarkFailed(ctx context.Context, eventID uuid.UUID, availableAt time.Time, cause string) error {
	const op = "OutboxRepository.MarkFailed"
	tag, err := dbFromContext(ctx, r.db).Exec(ctx, `
		UPDATE outbox_events
		SET attempts=attempts+1, last_error=$2, available_at=$3, locked_until=NULL
		WHERE id=$1 AND published_at IS NULL`, eventID, cause, availableAt)
	if err != nil {
		return wrap(op, err)
	}
	if tag.RowsAffected() == 0 {
		return wrap(op, pgx.ErrNoRows)
	}
	return nil
}
