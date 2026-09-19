package usecase

import (
	"context"
	"fmt"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
)

type relationshipUseCase struct {
	repository repository.RelationshipRepository
	outbox     repository.OutboxRepository
	tx         repository.TransactionManager
	profiles   ProfileProvider
}

func NewRelationshipUseCase(
	relationships repository.RelationshipRepository,
	outbox repository.OutboxRepository,
	tx repository.TransactionManager,
	profiles ProfileProvider,
) RelationshipUseCase {
	return &relationshipUseCase{repository: relationships, outbox: outbox, tx: tx, profiles: profiles}
}

func (uc *relationshipUseCase) Create(ctx context.Context, cmd CreateRelationshipCommand) (*entities.CoachAthleteRelationship, error) {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeCoach); err != nil {
		return nil, err
	}
	if cmd.AthleteID == uuid.Nil {
		return nil, invalid("athlete_id", "must be set")
	}
	if cmd.ActorID == cmd.AthleteID {
		return nil, invalid("athlete_id", "must differ from coach_id")
	}
	profileType, err := uc.profiles.GetProfileType(ctx, cmd.AthleteID)
	if err != nil {
		return nil, operationError("ProfileProvider.GetProfileType", err)
	}
	if profileType != ProfileTypeAthlete {
		return nil, operationError("RelationshipUseCase.Create", fmt.Errorf("%w: target user is not an athlete", ErrInvalidProfileType))
	}

	now := time.Now().UTC()
	relationship := &entities.CoachAthleteRelationship{
		ID:        uuid.New(),
		CoachID:   cmd.ActorID,
		AthleteID: cmd.AthleteID,
		Status:    entities.StatusRelActive,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = runTransaction(ctx, uc.tx, "RelationshipUseCase.Create", func(txCtx context.Context) error {
		active, err := uc.repository.IsActive(txCtx, cmd.ActorID, cmd.AthleteID)
		if err != nil {
			return operationError("RelationshipRepository.IsActive", err)
		}
		if active {
			return ErrRelationshipAlreadyExists
		}
		if err := uc.repository.Create(txCtx, relationship); err != nil {
			return operationError("RelationshipRepository.Create", err)
		}

		return createEvent(txCtx, uc.outbox, "coach_athlete.created", "coach_athlete_relationship", relationship.ID, map[string]any{
			"relationship_id": relationship.ID,
			"coach_id":        relationship.CoachID,
			"athlete_id":      relationship.AthleteID,
		}, now)
	})
	if err != nil {
		return nil, err
	}

	return relationship, nil
}

func (uc *relationshipUseCase) Terminate(ctx context.Context, cmd TerminateRelationshipCommand) error {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeCoach); err != nil {
		return err
	}
	if cmd.RelationshipID == uuid.Nil {
		return invalid("relationship_id", "must be set")
	}
	if err := validExpectedVersion(cmd.ExpectedVersion); err != nil {
		return err
	}

	now := time.Now().UTC()
	return runTransaction(ctx, uc.tx, "RelationshipUseCase.Terminate", func(txCtx context.Context) error {
		relationship, err := uc.repository.GetByID(txCtx, cmd.RelationshipID)
		if err != nil {
			return operationError("RelationshipRepository.GetByID", err)
		}
		if relationship == nil {
			return entities.ErrRelationshipNotFound
		}
		if relationship.CoachID != cmd.ActorID {
			return ErrForbidden
		}
		if relationship.Status != entities.StatusRelActive {
			return entities.ErrRelationshipNotActive
		}
		if err := versionMatches(relationship.Version, cmd.ExpectedVersion); err != nil {
			return err
		}
		if err := uc.repository.Terminate(txCtx, relationship.ID, cmd.ExpectedVersion); err != nil {
			return operationError("RelationshipRepository.Terminate", err)
		}

		return createEvent(txCtx, uc.outbox, "coach_athlete.terminated", "coach_athlete_relationship", relationship.ID, map[string]any{
			"relationship_id": relationship.ID,
			"coach_id":        relationship.CoachID,
			"athlete_id":      relationship.AthleteID,
		}, now)
	})
}

func (uc *relationshipUseCase) ListAthletes(ctx context.Context, actorID uuid.UUID) ([]uuid.UUID, error) {
	if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeCoach); err != nil {
		return nil, err
	}

	athletes, err := uc.repository.ListAthletes(ctx, actorID)
	if err != nil {
		return nil, operationError("RelationshipRepository.ListAthletes", err)
	}

	return athletes, nil
}

func (uc *relationshipUseCase) ListCoaches(ctx context.Context, actorID uuid.UUID) ([]uuid.UUID, error) {
	if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeAthlete); err != nil {
		return nil, err
	}

	coaches, err := uc.repository.ListCoaches(ctx, actorID)
	if err != nil {
		return nil, operationError("RelationshipRepository.ListCoaches", err)
	}

	return coaches, nil
}
