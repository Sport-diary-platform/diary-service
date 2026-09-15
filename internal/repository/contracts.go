package repository

import (
	"context"
	"diary-service/internal/entities"
	"time"

	"github.com/google/uuid"
)

type RelationshipRepository interface {
    Create(ctx context.Context, relationship *entities.CoachAthleteRelationship) error
    Get(ctx context.Context, coachID, athleteID uuid.UUID) (*entities.CoachAthleteRelationship, error)
    IsActive(ctx context.Context, coachID, athleteID uuid.UUID) (bool, error)
    Terminate(ctx context.Context, relationshipID uuid.UUID) error
    ListAthletes(ctx context.Context, coachID uuid.UUID) ([]uuid.UUID, error)
    ListCoaches(ctx context.Context, athleteID uuid.UUID) ([]uuid.UUID, error)
}

type TrainingPlanRepository interface {
    Create(ctx context.Context, plan *entities.TrainingPlan) error
    GetByID(ctx context.Context, planID uuid.UUID) (*entities.TrainingPlan, error)
    Update(ctx context.Context, plan *entities.TrainingPlan) error
    ListByAthlete(ctx context.Context, athleteID uuid.UUID) ([]entities.TrainingPlan, error)
    ListByCoach(ctx context.Context, coachID uuid.UUID) ([]entities.TrainingPlan, error)
}

type WorkoutRepository interface {
    Create(ctx context.Context, workout *entities.Workout) error
    GetByID(ctx context.Context, workoutID uuid.UUID) (*entities.Workout, error)
    Update(ctx context.Context, workout *entities.Workout) error
    Delete(ctx context.Context, workoutID uuid.UUID) error
    ListByAthlete(ctx context.Context, athleteID uuid.UUID, from, to time.Time) ([]entities.Workout, error)
    ListByCoach(ctx context.Context, coachID uuid.UUID, from, to time.Time) ([]entities.Workout, error)
}

type WorkoutResultRepository interface {
    Create(ctx context.Context, result *entities.WorkoutResult) error
    GetByWorkoutID(ctx context.Context, workoutID uuid.UUID) (*entities.WorkoutResult, error)
    Update(ctx context.Context, result *entities.WorkoutResult) error
    ExistsByWorkoutID(ctx context.Context, workoutID uuid.UUID) (bool, error)
}

type DailyCheckInRepository interface {
    Create(ctx context.Context, checkIn *entities.DailyCheckIn) error
    GetByDate(ctx context.Context, athleteID uuid.UUID, date time.Time) (*entities.DailyCheckIn, error)
    Update(ctx context.Context, checkIn *entities.DailyCheckIn) error
    List(ctx context.Context, athleteID uuid.UUID, from, to time.Time) ([]entities.DailyCheckIn, error)
}

type GoalRepository interface {
    Create(ctx context.Context, goal *entities.Goal) error
    GetByID(ctx context.Context, goalID uuid.UUID) (*entities.Goal, error)
    Update(ctx context.Context, goal *entities.Goal) error
    ListByAthlete(ctx context.Context, athleteID uuid.UUID) ([]entities.Goal, error)
}

type CommentRepository interface {
    Create(ctx context.Context, comment *entities.WorkoutComment) error
    ListByWorkout(ctx context.Context, workoutID uuid.UUID) ([]entities.WorkoutComment, error)
    Update(ctx context.Context, comment *entities.WorkoutComment) error
}

type OutboxRepository interface {
    Create(ctx context.Context, event *entities.OutboxEvent) error
    GetUnpublished(ctx context.Context, limit int) ([]entities.OutboxEvent, error)
    MarkPublished(ctx context.Context, eventID uuid.UUID) error
}

type TransactionManager interface {
    WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}