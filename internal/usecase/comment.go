package usecase

import (
	"context"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
)

type commentUseCase struct {
	repository    repository.CommentRepository
	workouts      repository.WorkoutRepository
	relationships repository.RelationshipRepository
	tx            repository.TransactionManager
	profiles      ProfileProvider
}

func NewCommentUseCase(
	comments repository.CommentRepository,
	workouts repository.WorkoutRepository,
	relationships repository.RelationshipRepository,
	tx repository.TransactionManager,
	profiles ProfileProvider,
) CommentUseCase {
	return &commentUseCase{
		repository: comments, workouts: workouts, relationships: relationships,
		tx: tx, profiles: profiles,
	}
}

func (uc *commentUseCase) Create(ctx context.Context, cmd CreateCommentCommand) (*entities.WorkoutComment, error) {
	if err := requireActor(cmd.ActorID); err != nil {
		return nil, err
	}
	if cmd.WorkoutID == uuid.Nil {
		return nil, invalid("workout_id", "must be set")
	}
	if err := validateRequiredString("text", cmd.Text); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	comment := &entities.WorkoutComment{
		ID: uuid.New(), WorkoutID: cmd.WorkoutID, AuthorID: cmd.ActorID, Text: cmd.Text,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	err := runTransaction(ctx, uc.tx, "CommentUseCase.Create", func(txCtx context.Context) error {
		workout, err := uc.workouts.GetByID(txCtx, cmd.WorkoutID)
		if err != nil {
			return operationError("WorkoutRepository.GetByID", err)
		}
		if workout == nil {
			return entities.ErrWorkoutNotFound
		}
		if err := uc.authorize(txCtx, cmd.ActorID, workout); err != nil {
			return err
		}
		if err := uc.repository.Create(txCtx, comment); err != nil {
			return operationError("CommentRepository.Create", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return comment, nil
}

func (uc *commentUseCase) ListByWorkout(ctx context.Context, actorID, workoutID uuid.UUID) ([]entities.WorkoutComment, error) {
	if err := requireActor(actorID); err != nil {
		return nil, err
	}
	if workoutID == uuid.Nil {
		return nil, invalid("workout_id", "must be set")
	}

	workout, err := uc.workouts.GetByID(ctx, workoutID)
	if err != nil {
		return nil, operationError("WorkoutRepository.GetByID", err)
	}
	if workout == nil {
		return nil, entities.ErrWorkoutNotFound
	}
	if err := uc.authorize(ctx, actorID, workout); err != nil {
		return nil, err
	}
	comments, err := uc.repository.ListByWorkout(ctx, workoutID)
	if err != nil {
		return nil, operationError("CommentRepository.ListByWorkout", err)
	}

	return comments, nil
}

func (uc *commentUseCase) Update(ctx context.Context, cmd UpdateCommentCommand) (*entities.WorkoutComment, error) {
	if err := requireActor(cmd.ActorID); err != nil {
		return nil, err
	}
	if cmd.WorkoutID == uuid.Nil {
		return nil, invalid("workout_id", "must be set")
	}
	if cmd.CommentID == uuid.Nil {
		return nil, invalid("comment_id", "must be set")
	}
	if err := validateRequiredString("text", cmd.Text); err != nil {
		return nil, err
	}
	if err := validExpectedVersion(cmd.ExpectedVersion); err != nil {
		return nil, err
	}

	var comment *entities.WorkoutComment
	now := time.Now().UTC()
	err := runTransaction(ctx, uc.tx, "CommentUseCase.Update", func(txCtx context.Context) error {
		workout, err := uc.workouts.GetByID(txCtx, cmd.WorkoutID)
		if err != nil {
			return operationError("WorkoutRepository.GetByID", err)
		}
		if workout == nil {
			return entities.ErrWorkoutNotFound
		}
		if err := uc.authorize(txCtx, cmd.ActorID, workout); err != nil {
			return err
		}
		comments, err := uc.repository.ListByWorkout(txCtx, cmd.WorkoutID)
		if err != nil {
			return operationError("CommentRepository.ListByWorkout", err)
		}
		for i := range comments {
			if comments[i].ID == cmd.CommentID {
				comment = &comments[i]
				break
			}
		}
		if comment == nil {
			return ErrCommentNotFound
		}
		if comment.WorkoutID != cmd.WorkoutID || comment.AuthorID != cmd.ActorID {
			return ErrForbidden
		}
		if err := versionMatches(comment.Version, cmd.ExpectedVersion); err != nil {
			return err
		}

		comment.Text = cmd.Text
		comment.UpdatedAt = now
		if err := uc.repository.Update(txCtx, comment); err != nil {
			return operationError("CommentRepository.Update", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return comment, nil
}

func (uc *commentUseCase) authorize(ctx context.Context, actorID uuid.UUID, workout *entities.Workout) error {
	if actorID == workout.AthleteID {
		return requireProfile(ctx, uc.profiles, actorID, ProfileTypeAthlete)
	}
	if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeCoach); err != nil {
		return err
	}

	return requireActiveRelationship(ctx, uc.relationships, actorID, workout.AthleteID)
}
