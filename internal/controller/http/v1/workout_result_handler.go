package v1

import (
	"diary-service/internal/controller/http/v1/request"
	"diary-service/internal/controller/http/v1/response"
	"diary-service/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

func (h *V1) CreateWorkoutResult(c fiber.Ctx) error {
	const function = "CreateWorkoutResult"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	workoutID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	var body request.CreateWorkoutResult
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	result, err := h.services.WorkoutResults.Create(c.Context(), usecase.CreateWorkoutResultCommand{
		ActorID: actorID, WorkoutID: workoutID, PerformedAt: body.PerformedAt, DurationSeconds: body.DurationSeconds,
		DistanceMeters: body.DistanceMeters, AvgHeartRate: body.AvgHeartRate, MaxHeartRate: body.MaxHeartRate,
		Calories: body.Calories, RPE: body.RPE, Feeling: body.Feeling, Comment: body.Comment,
		Exercises: exerciseResults(body.Exercises), ExpectedWorkoutVersion: body.ExpectedWorkoutVersion,
	})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("workout result"))
	}
	return c.Status(fiber.StatusCreated).JSON(response.WorkoutResultFromEntity(result))
}

func (h *V1) GetWorkoutResult(c fiber.Ctx) error {
	const function = "GetWorkoutResult"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	workoutID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	result, err := h.services.WorkoutResults.GetByWorkoutID(c.Context(), actorID, workoutID)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("workout result"))
	}
	return c.JSON(response.WorkoutResultFromEntity(result))
}

func (h *V1) UpdateWorkoutResult(c fiber.Ctx) error {
	const function = "UpdateWorkoutResult"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	workoutID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	var body request.UpdateWorkoutResult
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	result, err := h.services.WorkoutResults.Update(c.Context(), usecase.UpdateWorkoutResultCommand{
		ActorID: actorID, WorkoutID: workoutID, PerformedAt: body.PerformedAt, DurationSeconds: body.DurationSeconds,
		DistanceMeters: body.DistanceMeters, AvgHeartRate: body.AvgHeartRate, MaxHeartRate: body.MaxHeartRate,
		Calories: body.Calories, RPE: body.RPE, Feeling: body.Feeling, Comment: body.Comment,
		Exercises: exerciseResults(body.Exercises), ExpectedVersion: body.ExpectedVersion,
	})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("workout result"))
	}
	return c.JSON(response.WorkoutResultFromEntity(result))
}

func exerciseResults(values []request.ExerciseResult) []usecase.ExerciseResultCommand {
	result := make([]usecase.ExerciseResultCommand, 0, len(values))
	for _, exercise := range values {
		result = append(result, usecase.ExerciseResultCommand{ExerciseID: exercise.ExerciseID, Actual: exercise.Actual, Comment: exercise.Comment})
	}
	return result
}
