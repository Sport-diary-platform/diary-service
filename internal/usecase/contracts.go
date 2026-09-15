package usecase

import (
	"context"
	"diary-service/internal/entities"
	"time"

	"github.com/google/uuid"
)


type RelationshipUseCase interface {
    Create(ctx context.Context, rel *entities.CoachAthleteRelationship) (*entities.CoachAthleteRelationship, error)
    Terminate(ctx context.Context, relationshipID uuid.UUID) error
    ListAthletes(ctx context.Context, coachID uuid.UUID) ([]uuid.UUID, error)
    ListCoaches(ctx context.Context, athleteID uuid.UUID) ([]uuid.UUID, error)
}

type WorkoutUseCase interface {
    Create(ctx context.Context, workout *entities.Workout) (*entities.Workout, error)
    GetByID(ctx context.Context, workoutID uuid.UUID) (*entities.Workout, error)
    Update(ctx context.Context, workout *entities.Workout) (*entities.Workout, error)
    Cancel(ctx context.Context, workoutID uuid.UUID) error
    ListByAthlete(ctx context.Context, athleteID uuid.UUID, from, to time.Time) ([]entities.Workout, error)
    ListByCoach(ctx context.Context, coachID uuid.UUID, from, to time.Time) ([]entities.Workout, error)
}

type WorkoutResultUseCase interface {
    Create(ctx context.Context, workoutResult *entities.WorkoutResult) (*entities.WorkoutResult, error)
    GetByWorkoutID(ctx context.Context, workoutID uuid.UUID) (*entities.WorkoutResult, error)
    Update(ctx context.Context, workoutResult *entities.WorkoutResult) (*entities.WorkoutResult, error)
}

type TrainingPlanUseCase interface {
    Create(ctx context.Context, trainingPlan *entities.TrainingPlan) (*entities.TrainingPlan, error)
    GetByID(ctx context.Context, planID uuid.UUID) (*entities.TrainingPlan, error)
    Update(ctx context.Context, trainingPlan *entities.TrainingPlan) (*entities.TrainingPlan, error)
    Activate(ctx context.Context, planID uuid.UUID) error
    Complete(ctx context.Context, planID uuid.UUID) error
    Cancel(ctx context.Context, planID uuid.UUID) error
}

type DailyCheckInUseCase interface {
    Create(ctx context.Context, dailyCheckIn *entities.DailyCheckIn) (*entities.DailyCheckIn, error)
    GetByDate(ctx context.Context, athleteID uuid.UUID, date time.Time) (*entities.DailyCheckIn, error)
    Update(ctx context.Context, dailyCheckIn *entities.DailyCheckIn) (*entities.DailyCheckIn, error)
    List(ctx context.Context, athleteID uuid.UUID, from, to time.Time) ([]entities.DailyCheckIn, error)
}

type GoalUseCase interface {
    Create(ctx context.Context, goal *entities.Goal) (*entities.Goal, error)
    GetByID(ctx context.Context, goalID uuid.UUID) (*entities.Goal, error)
    Update(ctx context.Context, goal *entities.Goal) (*entities.Goal, error)
    Complete(ctx context.Context, goalID uuid.UUID) error
    Cancel(ctx context.Context, goalID uuid.UUID) error
}

type CommentUseCase interface {
    Create(ctx context.Context, comment *entities.WorkoutComment) (*entities.WorkoutComment, error)
    ListByWorkout(ctx context.Context, workoutID uuid.UUID) ([]entities.WorkoutComment, error)
    Update(ctx context.Context, comment *entities.WorkoutComment) (*entities.WorkoutComment, error)
}