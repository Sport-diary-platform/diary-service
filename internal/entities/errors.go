package entities

import "errors"

var (
	ErrInvalidInput               = errors.New("invalid input")
	ErrForbidden                  = errors.New("forbidden")
	ErrInvalidProfileType         = errors.New("invalid profile type")
	ErrWorkoutNotFound            = errors.New("workout not found")
	ErrWorkoutCancelled           = errors.New("workout is cancelled")
	ErrWorkoutCompleted           = errors.New("workout is completed")
	ErrWorkoutHasResult           = errors.New("workout has a result")
	ErrWorkoutResultAlreadyExists = errors.New("workout result already exists")
	ErrWorkoutResultNotFound      = errors.New("workout result not found")
	ErrInvalidRPE                 = errors.New("invalid rpe")
	ErrInvalidFeeling             = errors.New("invalid feeling")
	ErrTrainingPlanNotFound       = errors.New("training plan not found")
	ErrInvalidTrainingPlanState   = errors.New("invalid training plan state")
	ErrRelationshipNotFound       = errors.New("coach athlete relationship not found")
	ErrRelationshipNotActive      = errors.New("coach athlete relationship is not active")
	ErrRelationshipAlreadyExists  = errors.New("active coach athlete relationship already exists")
	ErrDailyCheckInAlreadyExists  = errors.New("daily check-in already exists")
	ErrDailyCheckInNotFound       = errors.New("daily check-in not found")
	ErrGoalNotFound               = errors.New("goal not found")
	ErrCommentNotFound            = errors.New("workout comment not found")
	ErrVersionConflict            = errors.New("version conflict")
)
