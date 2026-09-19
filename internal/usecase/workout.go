package usecase

import (
	"context"
	"encoding/json"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
)

type workoutUseCase struct {
	repository    repository.WorkoutRepository
	results       repository.WorkoutResultRepository
	plans         repository.TrainingPlanRepository
	relationships repository.RelationshipRepository
	outbox        repository.OutboxRepository
	tx            repository.TransactionManager
	profiles      ProfileProvider
}

func NewWorkoutUseCase(
	workouts repository.WorkoutRepository,
	results repository.WorkoutResultRepository,
	plans repository.TrainingPlanRepository,
	relationships repository.RelationshipRepository,
	outbox repository.OutboxRepository,
	tx repository.TransactionManager,
	profiles ProfileProvider,
) WorkoutUseCase {
	return &workoutUseCase{
		repository: workouts, results: results, plans: plans, relationships: relationships,
		outbox: outbox, tx: tx, profiles: profiles,
	}
}

func validateWorkout(title, sportType string, scheduledAt time.Time, estimatedDuration *int, blocks []WorkoutBlockCommand) error {
	if err := validateRequiredString("title", title); err != nil {
		return err
	}
	if err := validateRequiredString("sport_type", sportType); err != nil {
		return err
	}
	if scheduledAt.IsZero() {
		return invalid("scheduled_at", "must be set")
	}
	if estimatedDuration != nil && *estimatedDuration <= 0 {
		return invalid("estimated_duration_seconds", "must be greater than zero")
	}

	blockPositions := make(map[int]struct{}, len(blocks))
	blockIDs := make(map[uuid.UUID]struct{}, len(blocks))
	exerciseIDs := make(map[uuid.UUID]struct{})
	for _, block := range blocks {
		if err := validateRequiredString("blocks.name", block.Name); err != nil {
			return err
		}
		switch block.Type {
		case entities.WorkoutBlockTypeWarmup, entities.WorkoutBlockTypeMain, entities.WorkoutBlockTypeCooldown, entities.WorkoutBlockTypeOther:
		default:
			return invalid("blocks.type", "unsupported value")
		}
		if block.Position < 0 {
			return invalid("blocks.position", "must not be negative")
		}
		if _, exists := blockPositions[block.Position]; exists {
			return invalid("blocks.position", "must be unique within workout")
		}
		blockPositions[block.Position] = struct{}{}
		if block.ID != nil {
			if *block.ID == uuid.Nil {
				return invalid("blocks.id", "must not be nil UUID")
			}
			if _, exists := blockIDs[*block.ID]; exists {
				return invalid("blocks.id", "must be unique")
			}
			blockIDs[*block.ID] = struct{}{}
		}

		exercisePositions := make(map[int]struct{}, len(block.Exercises))
		for _, exercise := range block.Exercises {
			if err := validateRequiredString("blocks.exercises.name", exercise.Name); err != nil {
				return err
			}
			if err := validateRequiredString("blocks.exercises.type", exercise.Type); err != nil {
				return err
			}
			if exercise.Position < 0 {
				return invalid("blocks.exercises.position", "must not be negative")
			}
			if _, exists := exercisePositions[exercise.Position]; exists {
				return invalid("blocks.exercises.position", "must be unique within block")
			}
			exercisePositions[exercise.Position] = struct{}{}
			if err := validateJSON("blocks.exercises.target", exercise.Target); err != nil {
				return err
			}
			if exercise.ID != nil {
				if *exercise.ID == uuid.Nil {
					return invalid("blocks.exercises.id", "must not be nil UUID")
				}
				if _, exists := exerciseIDs[*exercise.ID]; exists {
					return invalid("blocks.exercises.id", "must be unique within block")
				}
				exerciseIDs[*exercise.ID] = struct{}{}
			}
		}
	}

	return nil
}

func buildWorkoutBlocks(workoutID uuid.UUID, commands []WorkoutBlockCommand, now time.Time) []entities.WorkoutBlock {
	blocks := make([]entities.WorkoutBlock, 0, len(commands))
	for _, command := range commands {
		blockID := uuid.New()
		if command.ID != nil {
			blockID = *command.ID
		}
		block := entities.WorkoutBlock{
			ID: blockID, WorkoutID: workoutID, Name: command.Name, Type: command.Type,
			Position: command.Position, CreatedAt: now, UpdatedAt: now,
			Exercises: make([]entities.WorkoutExercise, 0, len(command.Exercises)),
		}
		for _, exerciseCommand := range command.Exercises {
			exerciseID := uuid.New()
			if exerciseCommand.ID != nil {
				exerciseID = *exerciseCommand.ID
			}
			block.Exercises = append(block.Exercises, entities.WorkoutExercise{
				ID: exerciseID, WorkoutBlockID: blockID, Name: exerciseCommand.Name,
				Type: exerciseCommand.Type, Position: exerciseCommand.Position,
				Target: cloneJSON(exerciseCommand.Target), CreatedAt: now, UpdatedAt: now,
			})
		}
		blocks = append(blocks, block)
	}

	return blocks
}

func cloneJSON(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

func (uc *workoutUseCase) Create(ctx context.Context, cmd CreateWorkoutCommand) (*entities.Workout, error) {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeCoach); err != nil {
		return nil, err
	}
	if cmd.AthleteID == uuid.Nil {
		return nil, invalid("athlete_id", "must be set")
	}
	if err := validateWorkout(cmd.Title, cmd.SportType, cmd.ScheduledAt, cmd.EstimatedDurationSeconds, cmd.Blocks); err != nil {
		return nil, err
	}
	if err := requireActiveRelationship(ctx, uc.relationships, cmd.ActorID, cmd.AthleteID); err != nil {
		return nil, err
	}
	if err := uc.validatePlan(ctx, cmd.TrainingPlanID, cmd.ActorID, cmd.AthleteID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	workout := &entities.Workout{
		ID: uuid.New(), TrainingPlanID: cmd.TrainingPlanID, CoachID: cmd.ActorID, AthleteID: cmd.AthleteID,
		Title: cmd.Title, Description: cmd.Description, SportType: cmd.SportType, ScheduledAt: cmd.ScheduledAt,
		EstimatedDurationSeconds: cmd.EstimatedDurationSeconds, Status: entities.StatusWorkoutPlanned,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	workout.Blocks = buildWorkoutBlocks(workout.ID, cmd.Blocks, now)

	err := runTransaction(ctx, uc.tx, "WorkoutUseCase.Create", func(txCtx context.Context) error {
		if err := uc.repository.Create(txCtx, workout); err != nil {
			return operationError("WorkoutRepository.Create", err)
		}

		return createEvent(txCtx, uc.outbox, "workout.created", "workout", workout.ID, workoutEventPayload(workout), now)
	})
	if err != nil {
		return nil, err
	}

	return workout, nil
}

func (uc *workoutUseCase) GetByID(ctx context.Context, actorID, workoutID uuid.UUID) (*entities.Workout, error) {
	if err := requireActor(actorID); err != nil {
		return nil, err
	}
	if workoutID == uuid.Nil {
		return nil, invalid("workout_id", "must be set")
	}

	workout, err := uc.repository.GetByID(ctx, workoutID)
	if err != nil {
		return nil, operationError("WorkoutRepository.GetByID", err)
	}
	if workout == nil {
		return nil, entities.ErrWorkoutNotFound
	}
	if err := uc.authorizeRead(ctx, actorID, workout); err != nil {
		return nil, err
	}

	return workout, nil
}

func (uc *workoutUseCase) Update(ctx context.Context, cmd UpdateWorkoutCommand) (*entities.Workout, error) {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeCoach); err != nil {
		return nil, err
	}
	if cmd.WorkoutID == uuid.Nil {
		return nil, invalid("workout_id", "must be set")
	}
	if err := validExpectedVersion(cmd.ExpectedVersion); err != nil {
		return nil, err
	}
	if err := validateWorkout(cmd.Title, cmd.SportType, cmd.ScheduledAt, cmd.EstimatedDurationSeconds, cmd.Blocks); err != nil {
		return nil, err
	}

	var workout *entities.Workout
	now := time.Now().UTC()
	err := runTransaction(ctx, uc.tx, "WorkoutUseCase.Update", func(txCtx context.Context) error {
		var err error
		workout, err = uc.repository.GetByID(txCtx, cmd.WorkoutID)
		if err != nil {
			return operationError("WorkoutRepository.GetByID", err)
		}
		if workout == nil {
			return entities.ErrWorkoutNotFound
		}
		if err := uc.authorizeCoach(txCtx, cmd.ActorID, workout); err != nil {
			return err
		}
		if err := ensureWorkoutPlanned(workout); err != nil {
			return err
		}
		if err := versionMatches(workout.Version, cmd.ExpectedVersion); err != nil {
			return err
		}
		exists, err := uc.results.ExistsByWorkoutID(txCtx, workout.ID)
		if err != nil {
			return operationError("WorkoutResultRepository.ExistsByWorkoutID", err)
		}
		if exists {
			return entities.ErrWorkoutHasResult
		}

		workout.Title = cmd.Title
		workout.Description = cmd.Description
		workout.SportType = cmd.SportType
		workout.ScheduledAt = cmd.ScheduledAt
		workout.EstimatedDurationSeconds = cmd.EstimatedDurationSeconds
		workout.Blocks = buildWorkoutBlocks(workout.ID, cmd.Blocks, now)
		workout.UpdatedAt = now
		if err := uc.repository.Update(txCtx, workout); err != nil {
			return operationError("WorkoutRepository.Update", err)
		}
		if err := createEvent(txCtx, uc.outbox, "workout.updated", "workout", workout.ID, workoutEventPayload(workout), now); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return workout, nil
}

func (uc *workoutUseCase) Cancel(ctx context.Context, cmd ChangeWorkoutStateCommand) error {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeCoach); err != nil {
		return err
	}
	if cmd.WorkoutID == uuid.Nil {
		return invalid("workout_id", "must be set")
	}
	if err := validExpectedVersion(cmd.ExpectedVersion); err != nil {
		return err
	}

	now := time.Now().UTC()
	return runTransaction(ctx, uc.tx, "WorkoutUseCase.Cancel", func(txCtx context.Context) error {
		workout, err := uc.repository.GetByID(txCtx, cmd.WorkoutID)
		if err != nil {
			return operationError("WorkoutRepository.GetByID", err)
		}
		if workout == nil {
			return entities.ErrWorkoutNotFound
		}
		if err := uc.authorizeCoach(txCtx, cmd.ActorID, workout); err != nil {
			return err
		}
		if err := ensureWorkoutPlanned(workout); err != nil {
			return err
		}
		if err := versionMatches(workout.Version, cmd.ExpectedVersion); err != nil {
			return err
		}

		if err := workout.Cancel(); err != nil {
			return err
		}
		workout.UpdatedAt = now
		if err := uc.repository.Update(txCtx, workout); err != nil {
			return operationError("WorkoutRepository.Update", err)
		}

		return createEvent(txCtx, uc.outbox, "workout.cancelled", "workout", workout.ID, workoutEventPayload(workout), now)
	})
}

func (uc *workoutUseCase) Delete(ctx context.Context, cmd ChangeWorkoutStateCommand) error {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeCoach); err != nil {
		return err
	}
	if cmd.WorkoutID == uuid.Nil {
		return invalid("workout_id", "must be set")
	}
	if err := validExpectedVersion(cmd.ExpectedVersion); err != nil {
		return err
	}

	now := time.Now().UTC()
	return runTransaction(ctx, uc.tx, "WorkoutUseCase.Delete", func(txCtx context.Context) error {
		workout, err := uc.repository.GetByID(txCtx, cmd.WorkoutID)
		if err != nil {
			return operationError("WorkoutRepository.GetByID", err)
		}
		if workout == nil {
			return entities.ErrWorkoutNotFound
		}
		if err := uc.authorizeCoach(txCtx, cmd.ActorID, workout); err != nil {
			return err
		}
		if err := ensureWorkoutPlanned(workout); err != nil {
			return err
		}
		if err := versionMatches(workout.Version, cmd.ExpectedVersion); err != nil {
			return err
		}
		exists, err := uc.results.ExistsByWorkoutID(txCtx, workout.ID)
		if err != nil {
			return operationError("WorkoutResultRepository.ExistsByWorkoutID", err)
		}
		if exists {
			return entities.ErrWorkoutHasResult
		}
		if err := uc.repository.Delete(txCtx, workout.ID, cmd.ExpectedVersion); err != nil {
			return operationError("WorkoutRepository.Delete", err)
		}

		return createEvent(txCtx, uc.outbox, "workout.deleted", "workout", workout.ID, workoutEventPayload(workout), now)
	})
}

func (uc *workoutUseCase) ListByAthlete(ctx context.Context, actorID, athleteID uuid.UUID, from, to time.Time) ([]entities.Workout, error) {
	if err := requireActor(actorID); err != nil {
		return nil, err
	}
	if athleteID == uuid.Nil {
		return nil, invalid("athlete_id", "must be set")
	}
	if err := validateTimeRange(from, to); err != nil {
		return nil, err
	}
	if actorID == athleteID {
		if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeAthlete); err != nil {
			return nil, err
		}
	} else {
		if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeCoach); err != nil {
			return nil, err
		}
		if err := requireActiveRelationship(ctx, uc.relationships, actorID, athleteID); err != nil {
			return nil, err
		}
	}

	workouts, err := uc.repository.ListByAthlete(ctx, athleteID, from, to)
	if err != nil {
		return nil, operationError("WorkoutRepository.ListByAthlete", err)
	}

	return workouts, nil
}

func (uc *workoutUseCase) ListByCoach(ctx context.Context, actorID uuid.UUID, from, to time.Time) ([]entities.Workout, error) {
	if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeCoach); err != nil {
		return nil, err
	}
	if err := validateTimeRange(from, to); err != nil {
		return nil, err
	}

	workouts, err := uc.repository.ListByCoach(ctx, actorID, from, to)
	if err != nil {
		return nil, operationError("WorkoutRepository.ListByCoach", err)
	}

	return workouts, nil
}

func (uc *workoutUseCase) validatePlan(ctx context.Context, planID *uuid.UUID, coachID, athleteID uuid.UUID) error {
	if planID == nil {
		return nil
	}
	if *planID == uuid.Nil {
		return invalid("training_plan_id", "must not be nil UUID")
	}

	plan, err := uc.plans.GetByID(ctx, *planID)
	if err != nil {
		return operationError("TrainingPlanRepository.GetByID", err)
	}
	if plan == nil {
		return entities.ErrTrainingPlanNotFound
	}
	if plan.CoachID != coachID || plan.AthleteID != athleteID {
		return ErrForbidden
	}
	if plan.Status == entities.StatusPlanCompleted || plan.Status == entities.StatusPlanCancelled {
		return entities.ErrInvalidTrainingPlanState
	}

	return nil
}

func (uc *workoutUseCase) authorizeRead(ctx context.Context, actorID uuid.UUID, workout *entities.Workout) error {
	if actorID == workout.AthleteID {
		return requireProfile(ctx, uc.profiles, actorID, ProfileTypeAthlete)
	}
	if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeCoach); err != nil {
		return err
	}

	return requireActiveRelationship(ctx, uc.relationships, actorID, workout.AthleteID)
}

func (uc *workoutUseCase) authorizeCoach(ctx context.Context, actorID uuid.UUID, workout *entities.Workout) error {
	if workout.CoachID != actorID {
		return ErrForbidden
	}

	return requireActiveRelationship(ctx, uc.relationships, workout.CoachID, workout.AthleteID)
}

func ensureWorkoutPlanned(workout *entities.Workout) error {
	switch workout.Status {
	case entities.StatusWorkoutPlanned:
		return nil
	case entities.StatusWorkoutCancelled:
		return entities.ErrWorkoutCancelled
	case entities.StatusWorkoutCompleted:
		return entities.ErrWorkoutCompleted
	default:
		return invalid("workout.status", "unsupported value")
	}
}

func workoutEventPayload(workout *entities.Workout) map[string]any {
	return map[string]any{
		"workout_id": workout.ID,
		"coach_id":   workout.CoachID,
		"athlete_id": workout.AthleteID,
		"status":     workout.Status,
	}
}
