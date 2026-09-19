package usecase

import (
	"context"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
)

type trainingPlanUseCase struct {
	repository    repository.TrainingPlanRepository
	relationships repository.RelationshipRepository
	outbox        repository.OutboxRepository
	tx            repository.TransactionManager
	profiles      ProfileProvider
}

func NewTrainingPlanUseCase(
	plans repository.TrainingPlanRepository,
	relationships repository.RelationshipRepository,
	outbox repository.OutboxRepository,
	tx repository.TransactionManager,
	profiles ProfileProvider,
) TrainingPlanUseCase {
	return &trainingPlanUseCase{repository: plans, relationships: relationships, outbox: outbox, tx: tx, profiles: profiles}
}

func validateTrainingPlan(name string, startDate entities.LocalDate, endDate *entities.LocalDate) error {
	if err := validateRequiredString("name", name); err != nil {
		return err
	}
	if err := validateLocalDate("start_date", startDate); err != nil {
		return err
	}
	if endDate != nil {
		if err := validateLocalDate("end_date", *endDate); err != nil {
			return err
		}
		if string(*endDate) < string(startDate) {
			return invalid("end_date", "must not be before start_date")
		}
	}

	return nil
}

func (uc *trainingPlanUseCase) Create(ctx context.Context, cmd CreateTrainingPlanCommand) (*entities.TrainingPlan, error) {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeCoach); err != nil {
		return nil, err
	}
	if cmd.AthleteID == uuid.Nil {
		return nil, invalid("athlete_id", "must be set")
	}
	if err := validateTrainingPlan(cmd.Name, cmd.StartDate, cmd.EndDate); err != nil {
		return nil, err
	}
	if err := requireActiveRelationship(ctx, uc.relationships, cmd.ActorID, cmd.AthleteID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	plan := &entities.TrainingPlan{
		ID:          uuid.New(),
		CoachID:     cmd.ActorID,
		AthleteID:   cmd.AthleteID,
		Name:        cmd.Name,
		Description: cmd.Description,
		StartDate:   cmd.StartDate,
		EndDate:     cmd.EndDate,
		Status:      entities.StatusPlanDraft,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := runTransaction(ctx, uc.tx, "TrainingPlanUseCase.Create", func(txCtx context.Context) error {
		if err := uc.repository.Create(txCtx, plan); err != nil {
			return operationError("TrainingPlanRepository.Create", err)
		}

		return createEvent(txCtx, uc.outbox, "training_plan.created", "training_plan", plan.ID, planEventPayload(plan), now)
	})
	if err != nil {
		return nil, err
	}

	return plan, nil
}

func (uc *trainingPlanUseCase) GetByID(ctx context.Context, actorID, planID uuid.UUID) (*entities.TrainingPlan, error) {
	if err := requireActor(actorID); err != nil {
		return nil, err
	}
	if planID == uuid.Nil {
		return nil, invalid("plan_id", "must be set")
	}

	plan, err := uc.repository.GetByID(ctx, planID)
	if err != nil {
		return nil, operationError("TrainingPlanRepository.GetByID", err)
	}
	if plan == nil {
		return nil, entities.ErrTrainingPlanNotFound
	}
	if err := uc.authorizeRead(ctx, actorID, plan); err != nil {
		return nil, err
	}

	return plan, nil
}

func (uc *trainingPlanUseCase) Update(ctx context.Context, cmd UpdateTrainingPlanCommand) (*entities.TrainingPlan, error) {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeCoach); err != nil {
		return nil, err
	}
	if cmd.PlanID == uuid.Nil {
		return nil, invalid("plan_id", "must be set")
	}
	if err := validateTrainingPlan(cmd.Name, cmd.StartDate, cmd.EndDate); err != nil {
		return nil, err
	}
	if err := validExpectedVersion(cmd.ExpectedVersion); err != nil {
		return nil, err
	}

	var plan *entities.TrainingPlan
	now := time.Now().UTC()
	err := runTransaction(ctx, uc.tx, "TrainingPlanUseCase.Update", func(txCtx context.Context) error {
		var err error
		plan, err = uc.repository.GetByID(txCtx, cmd.PlanID)
		if err != nil {
			return operationError("TrainingPlanRepository.GetByID", err)
		}
		if plan == nil {
			return entities.ErrTrainingPlanNotFound
		}
		if err := uc.authorizeCoach(txCtx, cmd.ActorID, plan); err != nil {
			return err
		}
		if plan.Status != entities.StatusPlanDraft && plan.Status != entities.StatusPlanActive {
			return entities.ErrInvalidTrainingPlanState
		}
		if err := versionMatches(plan.Version, cmd.ExpectedVersion); err != nil {
			return err
		}

		plan.Name = cmd.Name
		plan.Description = cmd.Description
		plan.StartDate = cmd.StartDate
		plan.EndDate = cmd.EndDate
		plan.UpdatedAt = now
		if err := uc.repository.Update(txCtx, plan); err != nil {
			return operationError("TrainingPlanRepository.Update", err)
		}
		if err := createEvent(txCtx, uc.outbox, "training_plan.updated", "training_plan", plan.ID, planEventPayload(plan), now); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return plan, nil
}

func (uc *trainingPlanUseCase) Activate(ctx context.Context, cmd ChangeTrainingPlanStateCommand) error {
	return uc.changeState(ctx, cmd, entities.StatusPlanDraft, entities.StatusPlanActive, "training_plan.activated")
}

func (uc *trainingPlanUseCase) Complete(ctx context.Context, cmd ChangeTrainingPlanStateCommand) error {
	return uc.changeState(ctx, cmd, entities.StatusPlanActive, entities.StatusPlanCompleted, "training_plan.completed")
}

func (uc *trainingPlanUseCase) Cancel(ctx context.Context, cmd ChangeTrainingPlanStateCommand) error {
	return uc.changeState(ctx, cmd, "", entities.StatusPlanCancelled, "training_plan.cancelled")
}

func (uc *trainingPlanUseCase) changeState(ctx context.Context, cmd ChangeTrainingPlanStateCommand, required, target entities.StatusPlan, eventType string) error {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeCoach); err != nil {
		return err
	}
	if cmd.PlanID == uuid.Nil {
		return invalid("plan_id", "must be set")
	}
	if err := validExpectedVersion(cmd.ExpectedVersion); err != nil {
		return err
	}

	now := time.Now().UTC()
	return runTransaction(ctx, uc.tx, "TrainingPlanUseCase.changeState", func(txCtx context.Context) error {
		plan, err := uc.repository.GetByID(txCtx, cmd.PlanID)
		if err != nil {
			return operationError("TrainingPlanRepository.GetByID", err)
		}
		if plan == nil {
			return entities.ErrTrainingPlanNotFound
		}
		if err := uc.authorizeCoach(txCtx, cmd.ActorID, plan); err != nil {
			return err
		}
		if required != "" {
			if plan.Status != required {
				return entities.ErrInvalidTrainingPlanState
			}
		} else if plan.Status != entities.StatusPlanDraft && plan.Status != entities.StatusPlanActive {
			return entities.ErrInvalidTrainingPlanState
		}
		if err := versionMatches(plan.Version, cmd.ExpectedVersion); err != nil {
			return err
		}

		plan.Status = target
		plan.UpdatedAt = now
		if err := uc.repository.Update(txCtx, plan); err != nil {
			return operationError("TrainingPlanRepository.Update", err)
		}

		return createEvent(txCtx, uc.outbox, eventType, "training_plan", plan.ID, planEventPayload(plan), now)
	})
}

func (uc *trainingPlanUseCase) authorizeRead(ctx context.Context, actorID uuid.UUID, plan *entities.TrainingPlan) error {
	if actorID == plan.AthleteID {
		return requireProfile(ctx, uc.profiles, actorID, ProfileTypeAthlete)
	}
	if actorID == plan.CoachID {
		if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeCoach); err != nil {
			return err
		}

		return requireActiveRelationship(ctx, uc.relationships, plan.CoachID, plan.AthleteID)
	}

	return ErrForbidden
}

func (uc *trainingPlanUseCase) authorizeCoach(ctx context.Context, actorID uuid.UUID, plan *entities.TrainingPlan) error {
	if plan.CoachID != actorID {
		return ErrForbidden
	}

	return requireActiveRelationship(ctx, uc.relationships, plan.CoachID, plan.AthleteID)
}

func planEventPayload(plan *entities.TrainingPlan) map[string]any {
	return map[string]any{
		"training_plan_id": plan.ID,
		"coach_id":         plan.CoachID,
		"athlete_id":       plan.AthleteID,
		"status":           plan.Status,
	}
}
