package entities

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWorkoutStateTransitions(t *testing.T) {
	t.Run("planned workout can be completed", func(t *testing.T) {
		workout := Workout{Status: StatusWorkoutPlanned}
		require.NoError(t, workout.Complete())
		require.Equal(t, StatusWorkoutCompleted, workout.Status)
		require.ErrorIs(t, workout.Cancel(), ErrWorkoutCompleted)
	})

	t.Run("cancelled workout is terminal", func(t *testing.T) {
		workout := Workout{Status: StatusWorkoutPlanned}
		require.NoError(t, workout.Cancel())
		require.Equal(t, StatusWorkoutCancelled, workout.Status)
		require.ErrorIs(t, workout.Complete(), ErrWorkoutCancelled)
	})
}

func TestWorkoutBlocks(t *testing.T) {
	workout := Workout{ID: uuid.New()}
	first := WorkoutBlock{ID: uuid.New(), Position: 0}
	require.NoError(t, workout.AddBlock(first))
	require.Equal(t, workout.ID, workout.Blocks[0].WorkoutID)
	require.True(t, errors.Is(workout.AddBlock(WorkoutBlock{Position: 0}), ErrInvalidInput))
	require.NoError(t, workout.RemoveBlock(first.ID))
	require.Empty(t, workout.Blocks)
}
