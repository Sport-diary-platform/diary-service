package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type StatusWorkout string
type WorkoutBlockType string

const (
	StatusWorkoutPlanned   StatusWorkout = "planned"
	StatusWorkoutCompleted StatusWorkout = "completed"
	StatusWorkoutCancelled StatusWorkout = "cancelled"
)

const (
	WorkoutBlockTypeWarmup   WorkoutBlockType = "warmup"
	WorkoutBlockTypeMain     WorkoutBlockType = "main"
	WorkoutBlockTypeCooldown WorkoutBlockType = "cooldown"
	WorkoutBlockTypeOther    WorkoutBlockType = "other"
)

type Workout struct {
	ID                       uuid.UUID
	TrainingPlanID           *uuid.UUID
	CoachID                  uuid.UUID
	AthleteID                uuid.UUID
	Title                    string
	Description              *string
	SportType                string
	ScheduledAt              time.Time
	EstimatedDurationSeconds *int
	Status                   StatusWorkout
	Blocks                   []WorkoutBlock
	Version                  int64
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type WorkoutBlock struct {
	ID        uuid.UUID
	WorkoutID uuid.UUID
	Name      string
	Type      WorkoutBlockType
	Position  int
	Exercises []WorkoutExercise
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WorkoutExercise struct {
	ID             uuid.UUID
	WorkoutBlockID uuid.UUID
	Name           string
	Type           string
	Position       int
	Target         json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type WorkoutResult struct {
	ID              uuid.UUID
	WorkoutID       uuid.UUID
	AthleteID       uuid.UUID
	PerformedAt     time.Time
	DurationSeconds *int
	DistanceMeters  *float64
	AvgHeartRate    *int
	MaxHeartRate    *int
	Calories        *int
	RPE             int
	Feeling         int
	Comment         *string
	ExerciseResults []WorkoutExerciseResult
	Version         int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type WorkoutExerciseResult struct {
	ID              uuid.UUID
	WorkoutResultID uuid.UUID
	ExerciseID      uuid.UUID
	Actual          json.RawMessage
	Comment         *string
	Version         int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (w *Workout) Cancel() error {
	if w.Status == StatusWorkoutCancelled {
		return ErrWorkoutCancelled
	}
	if w.Status == StatusWorkoutCompleted {
		return ErrWorkoutCompleted
	}
	w.Status = StatusWorkoutCancelled
	return nil
}

func (w *Workout) Complete() error {
	if w.Status == StatusWorkoutCancelled {
		return ErrWorkoutCancelled
	}
	if w.Status == StatusWorkoutCompleted {
		return ErrWorkoutCompleted
	}
	w.Status = StatusWorkoutCompleted
	return nil
}

func (w *Workout) AddBlock(block WorkoutBlock) error {
	for _, current := range w.Blocks {
		if current.Position == block.Position {
			return ErrInvalidInput
		}
	}
	block.WorkoutID = w.ID
	w.Blocks = append(w.Blocks, block)
	return nil
}

func (w *Workout) RemoveBlock(blockID uuid.UUID) error {
	for i := range w.Blocks {
		if w.Blocks[i].ID == blockID {
			w.Blocks = append(w.Blocks[:i], w.Blocks[i+1:]...)
			return nil
		}
	}
	return ErrInvalidInput
}

type WorkoutComment struct {
	ID        uuid.UUID
	WorkoutID uuid.UUID
	AuthorID  uuid.UUID
	Text      string
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
