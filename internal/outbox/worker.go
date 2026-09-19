package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/repository"
	"diary-service/pkg/kafka"
	"diary-service/pkg/logger"
)

type Producer interface {
	Produce(ctx context.Context, messages ...*kafka.Message) error
}

type Topics struct {
	CoachAthlete  string
	TrainingPlan  string
	Workout       string
	WorkoutResult string
	DailyCheckIn  string
	Goal          string
}

type Worker struct {
	repository   repository.OutboxRepository
	producer     Producer
	logger       logger.Interface
	topics       Topics
	pollInterval time.Duration
	lease        time.Duration
	batchSize    int
}

func NewWorker(repo repository.OutboxRepository, producer Producer, log logger.Interface, topics Topics, pollInterval, lease time.Duration, batchSize int) *Worker {
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	if lease <= 0 {
		lease = 30 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &Worker{repository: repo, producer: producer, logger: log, topics: topics, pollInterval: pollInterval, lease: lease, batchSize: batchSize}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		if err := w.publishBatch(ctx); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.ErrorDetails("outbox", "Run", "publish batch", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) publishBatch(ctx context.Context) error {
	events, err := w.repository.ClaimUnpublished(ctx, w.batchSize, time.Now().UTC().Add(w.lease))
	if err != nil {
		return errors.Join(errors.New("outbox - publishBatch - ClaimUnpublished"), err)
	}
	for i := range events {
		if err := w.publish(ctx, &events[i]); err != nil {
			backoff := time.Duration(math.Min(math.Pow(2, float64(events[i].Attempts)), 300)) * time.Second
			if markErr := w.repository.MarkFailed(ctx, events[i].ID, time.Now().UTC().Add(backoff), err.Error()); markErr != nil {
				w.logger.ErrorDetails("outbox", "publishBatch", "mark failed", markErr)
			}
			w.logger.ErrorDetails("outbox", "publishBatch", "produce", err)
			continue
		}
		if err := w.repository.MarkPublished(ctx, events[i].ID, time.Now().UTC()); err != nil {
			w.logger.ErrorDetails("outbox", "publishBatch", "mark published", err)
		}
	}
	return nil
}

func (w *Worker) publish(ctx context.Context, event *entities.OutboxEvent) error {
	topic := w.topic(event.EventType)
	if topic == "" {
		return errors.New("outbox - publish - topic is not configured")
	}
	envelope, err := json.Marshal(struct {
		EventID       string          `json:"event_id"`
		EventType     string          `json:"event_type"`
		EventVersion  int             `json:"event_version"`
		OccurredAt    time.Time       `json:"occurred_at"`
		AggregateType string          `json:"aggregate_type"`
		AggregateID   string          `json:"aggregate_id"`
		Payload       json.RawMessage `json:"payload"`
	}{event.ID.String(), event.EventType, event.EventVersion, event.CreatedAt, event.AggregateType, event.AggregateID.String(), event.Payload})
	if err != nil {
		return errors.Join(errors.New("outbox - publish - Marshal"), err)
	}
	return w.producer.Produce(ctx, &kafka.Message{Key: []byte(event.AggregateID.String()), Data: envelope, Topic: topic})
}

func (w *Worker) topic(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "coach_athlete."):
		return w.topics.CoachAthlete
	case strings.HasPrefix(eventType, "training_plan."):
		return w.topics.TrainingPlan
	case strings.HasPrefix(eventType, "workout_result."):
		return w.topics.WorkoutResult
	case strings.HasPrefix(eventType, "workout."):
		return w.topics.Workout
	case strings.HasPrefix(eventType, "daily_checkin."):
		return w.topics.DailyCheckIn
	case strings.HasPrefix(eventType, "goal."):
		return w.topics.Goal
	default:
		return ""
	}
}
