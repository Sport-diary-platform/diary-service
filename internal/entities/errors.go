package entities

import "errors"

var (
	ErrWorkoutNotFound            = errors.New("workout not found")
	ErrWorkoutCancelled           = errors.New("workout is cancelled")
	ErrWorkoutResultAlreadyExists = errors.New("workout result already exists")
	ErrWorkoutResultNotFound      = errors.New("workout result not found")
	ErrInvalidRPE                 = errors.New("invalid rpe")
	ErrInvalidFeeling             = errors.New("invalid feeling")
	ErrTrainingPlanNotFound       = errors.New("training plan not found")
	ErrInvalidTrainingPlanState   = errors.New("invalid training plan state")
	ErrRelationshipNotFound       = errors.New("coach athlete relationship not found")
	ErrRelationshipNotActive      = errors.New("coach athlete relationship is not active")
	ErrDailyCheckInAlreadyExists  = errors.New("daily check-in already exists")
	ErrDailyCheckInNotFound       = errors.New("daily check-in not found")
	ErrGoalNotFound               = errors.New("goal not found")
)
