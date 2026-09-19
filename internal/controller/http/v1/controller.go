package v1

import (
	"diary-service/internal/usecase"
	"diary-service/pkg/logger"

	"github.com/go-playground/validator/v10"
)

type Services struct {
	Relationships  usecase.RelationshipUseCase
	TrainingPlans  usecase.TrainingPlanUseCase
	Workouts       usecase.WorkoutUseCase
	WorkoutResults usecase.WorkoutResultUseCase
	CheckIns       usecase.DailyCheckInUseCase
	Goals          usecase.GoalUseCase
	Comments       usecase.CommentUseCase
}

type V1 struct {
	services  *Services
	logger    logger.Interface
	validator *validator.Validate
}

func New(services *Services, log logger.Interface) *V1 {
	if services == nil {
		services = &Services{}
	}
	return &V1{services: services, logger: log, validator: validator.New()}
}
