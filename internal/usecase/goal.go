package usecase

import (
	"context"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
)

type goalUseCase struct {
	repository    repository.GoalRepository
	relationships repository.RelationshipRepository
	outbox        repository.OutboxRepository
	tx            repository.TransactionManager
	profiles      ProfileProvider
}

func NewGoalUseCase(
	goals repository.GoalRepository,
	relationships repository.RelationshipRepository,
	outbox repository.OutboxRepository,
	tx repository.TransactionManager,
	profiles ProfileProvider,
) GoalUseCase {
	return &goalUseCase{repository: goals, relationships: relationships, outbox: outbox, tx: tx, profiles: profiles}
}

func validateGoal(title string, goalType entities.GoalType, target []byte, deadline *entities.LocalDate) error {
	if err := validateRequiredString("title", title); err != nil {
		return err
	}
	switch goalType {
	case entities.GoalTypePerformance, entities.GoalTypeWeight, entities.GoalTypeStrength,
		entities.GoalTypeDistance, entities.GoalTypeTime, entities.GoalTypeCustom:
	default:
		return invalid("type", "unsupported value")
	}
	if err := validateJSON("target", target); err != nil {
		return err
	}
	if deadline != nil {
		if err := validateLocalDate("deadline", *deadline); err != nil {
			return err
		}
	}

	return nil
}

func (uc *goalUseCase) Create(ctx context.Context, cmd CreateGoalCommand) (*entities.Goal, error) {
	if err := requireActor(cmd.ActorID); err != nil {
		return nil, err
	}
	if cmd.AthleteID == uuid.Nil {
		return nil, invalid("athlete_id", "must be set")
	}
	if err := validateGoal(cmd.Title, cmd.Type, cmd.Target, cmd.Deadline); err != nil {
		return nil, err
	}

	profileType, err := uc.profiles.GetProfileType(ctx, cmd.ActorID)
	if err != nil {
		return nil, operationError("ProfileProvider.GetProfileType", err)
	}
	var coachID *uuid.UUID
	switch profileType {
	case ProfileTypeAthlete:
		if cmd.ActorID != cmd.AthleteID {
			return nil, ErrForbidden
		}
	case ProfileTypeCoach:
		if err := requireActiveRelationship(ctx, uc.relationships, cmd.ActorID, cmd.AthleteID); err != nil {
			return nil, err
		}
		id := cmd.ActorID
		coachID = &id
	default:
		return nil, ErrInvalidProfileType
	}

	now := time.Now().UTC()
	goal := &entities.Goal{
		ID: uuid.New(), AthleteID: cmd.AthleteID, CoachID: coachID,
		Title: cmd.Title, Description: cmd.Description, Type: cmd.Type, Target: cloneJSON(cmd.Target),
		Deadline: cmd.Deadline, Status: entities.GoalStatusActive, Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	err = runTransaction(ctx, uc.tx, "GoalUseCase.Create", func(txCtx context.Context) error {
		if err := uc.repository.Create(txCtx, goal); err != nil {
			return operationError("GoalRepository.Create", err)
		}

		return createEvent(txCtx, uc.outbox, "goal.created", "goal", goal.ID, goalEventPayload(goal), now)
	})
	if err != nil {
		return nil, err
	}

	return goal, nil
}

func (uc *goalUseCase) GetByID(ctx context.Context, actorID, goalID uuid.UUID) (*entities.Goal, error) {
	if err := requireActor(actorID); err != nil {
		return nil, err
	}
	if goalID == uuid.Nil {
		return nil, invalid("goal_id", "must be set")
	}

	goal, err := uc.repository.GetByID(ctx, goalID)
	if err != nil {
		return nil, operationError("GoalRepository.GetByID", err)
	}
	if goal == nil {
		return nil, entities.ErrGoalNotFound
	}
	if err := uc.authorize(ctx, actorID, goal); err != nil {
		return nil, err
	}

	return goal, nil
}

func (uc *goalUseCase) Update(ctx context.Context, cmd UpdateGoalCommand) (*entities.Goal, error) {
	if err := requireActor(cmd.ActorID); err != nil {
		return nil, err
	}
	if cmd.GoalID == uuid.Nil {
		return nil, invalid("goal_id", "must be set")
	}
	if err := validateGoal(cmd.Title, cmd.Type, cmd.Target, cmd.Deadline); err != nil {
		return nil, err
	}
	if err := validExpectedVersion(cmd.ExpectedVersion); err != nil {
		return nil, err
	}

	var goal *entities.Goal
	now := time.Now().UTC()
	err := runTransaction(ctx, uc.tx, "GoalUseCase.Update", func(txCtx context.Context) error {
		var err error
		goal, err = uc.repository.GetByID(txCtx, cmd.GoalID)
		if err != nil {
			return operationError("GoalRepository.GetByID", err)
		}
		if goal == nil {
			return entities.ErrGoalNotFound
		}
		if err := uc.authorize(txCtx, cmd.ActorID, goal); err != nil {
			return err
		}
		if goal.Status != entities.GoalStatusActive {
			return invalid("goal.status", "only an active goal can be updated")
		}
		if err := versionMatches(goal.Version, cmd.ExpectedVersion); err != nil {
			return err
		}

		goal.Title = cmd.Title
		goal.Description = cmd.Description
		goal.Type = cmd.Type
		goal.Target = cloneJSON(cmd.Target)
		goal.Deadline = cmd.Deadline
		goal.UpdatedAt = now
		if err := uc.repository.Update(txCtx, goal); err != nil {
			return operationError("GoalRepository.Update", err)
		}

		return createEvent(txCtx, uc.outbox, "goal.updated", "goal", goal.ID, goalEventPayload(goal), now)
	})
	if err != nil {
		return nil, err
	}

	return goal, nil
}

func (uc *goalUseCase) Complete(ctx context.Context, cmd ChangeGoalStateCommand) error {
	return uc.changeState(ctx, cmd, entities.GoalStatusCompleted, "goal.completed")
}

func (uc *goalUseCase) Cancel(ctx context.Context, cmd ChangeGoalStateCommand) error {
	return uc.changeState(ctx, cmd, entities.GoalStatusCancelled, "goal.cancelled")
}

func (uc *goalUseCase) changeState(ctx context.Context, cmd ChangeGoalStateCommand, status entities.GoalStatus, eventType string) error {
	if err := requireActor(cmd.ActorID); err != nil {
		return err
	}
	if cmd.GoalID == uuid.Nil {
		return invalid("goal_id", "must be set")
	}
	if err := validExpectedVersion(cmd.ExpectedVersion); err != nil {
		return err
	}

	now := time.Now().UTC()
	return runTransaction(ctx, uc.tx, "GoalUseCase.changeState", func(txCtx context.Context) error {
		goal, err := uc.repository.GetByID(txCtx, cmd.GoalID)
		if err != nil {
			return operationError("GoalRepository.GetByID", err)
		}
		if goal == nil {
			return entities.ErrGoalNotFound
		}
		if err := uc.authorize(txCtx, cmd.ActorID, goal); err != nil {
			return err
		}
		if goal.Status != entities.GoalStatusActive {
			return invalid("goal.status", "only an active goal can change state")
		}
		if err := versionMatches(goal.Version, cmd.ExpectedVersion); err != nil {
			return err
		}

		goal.Status = status
		goal.UpdatedAt = now
		if err := uc.repository.Update(txCtx, goal); err != nil {
			return operationError("GoalRepository.Update", err)
		}

		return createEvent(txCtx, uc.outbox, eventType, "goal", goal.ID, goalEventPayload(goal), now)
	})
}

func (uc *goalUseCase) List(ctx context.Context, actorID uuid.UUID) ([]entities.Goal, error) {
	if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeAthlete); err != nil {
		return nil, err
	}

	goals, err := uc.repository.ListByAthlete(ctx, actorID)
	if err != nil {
		return nil, operationError("GoalRepository.ListByAthlete", err)
	}

	return goals, nil
}

func (uc *goalUseCase) authorize(ctx context.Context, actorID uuid.UUID, goal *entities.Goal) error {
	if actorID == goal.AthleteID {
		return requireProfile(ctx, uc.profiles, actorID, ProfileTypeAthlete)
	}
	if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeCoach); err != nil {
		return err
	}

	return requireActiveRelationship(ctx, uc.relationships, actorID, goal.AthleteID)
}

func goalEventPayload(goal *entities.Goal) map[string]any {
	return map[string]any{
		"goal_id":    goal.ID,
		"athlete_id": goal.AthleteID,
		"coach_id":   goal.CoachID,
		"status":     goal.Status,
	}
}
