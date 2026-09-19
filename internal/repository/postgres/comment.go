package postgres

import (
	"context"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
)

type commentRepository struct{ db DB }

func NewCommentRepository(db DB) repository.CommentRepository { return &commentRepository{db: db} }

func (r *commentRepository) Create(ctx context.Context, comment *entities.WorkoutComment) error {
	const op = "CommentRepository.Create"
	_, err := dbFromContext(ctx, r.db).Exec(ctx, `
		INSERT INTO workout_comments (id, workout_id, author_id, text, version, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, comment.ID, comment.WorkoutID, comment.AuthorID,
		comment.Text, comment.Version, comment.CreatedAt, comment.UpdatedAt)
	return wrap(op, err)
}

func (r *commentRepository) ListByWorkout(ctx context.Context, workoutID uuid.UUID) ([]entities.WorkoutComment, error) {
	const op = "CommentRepository.ListByWorkout"
	rows, err := dbFromContext(ctx, r.db).Query(ctx, `
		SELECT id, workout_id, author_id, text, version, created_at, updated_at
		FROM workout_comments WHERE workout_id=$1 ORDER BY created_at, id`, workoutID)
	if err != nil {
		return nil, wrap(op, err)
	}
	defer rows.Close()

	comments := make([]entities.WorkoutComment, 0)
	for rows.Next() {
		var comment entities.WorkoutComment
		if err = rows.Scan(&comment.ID, &comment.WorkoutID, &comment.AuthorID, &comment.Text,
			&comment.Version, &comment.CreatedAt, &comment.UpdatedAt); err != nil {
			return nil, wrap(op, err)
		}
		comments = append(comments, comment)
	}
	if err = rows.Err(); err != nil {
		return nil, wrap(op, err)
	}
	return comments, nil
}

func (r *commentRepository) Update(ctx context.Context, comment *entities.WorkoutComment) error {
	const op = "CommentRepository.Update"
	tag, err := dbFromContext(ctx, r.db).Exec(ctx, `
		UPDATE workout_comments SET text=$2, version=version+1, updated_at=$3
		WHERE id=$1 AND version=$4`, comment.ID, comment.Text, comment.UpdatedAt, comment.Version)
	if err != nil {
		return wrap(op, err)
	}
	if tag.RowsAffected() == 0 {
		return versionConflict(op)
	}
	comment.Version++
	return nil
}
