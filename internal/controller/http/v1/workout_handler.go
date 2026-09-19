package v1

import (
	"errors"

	"diary-service/internal/controller/http/v1/request"
	"diary-service/internal/controller/http/v1/response"
	"diary-service/internal/entities"
	"diary-service/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (h *V1) CreateWorkout(c fiber.Ctx) error {
	const function = "CreateWorkout"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	var body request.CreateWorkout
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	result, err := h.services.Workouts.Create(c.Context(), usecase.CreateWorkoutCommand{
		ActorID: actorID, AthleteID: body.AthleteID, TrainingPlanID: body.TrainingPlanID,
		Title: body.Title, Description: body.Description, SportType: body.SportType, ScheduledAt: body.ScheduledAt,
		EstimatedDurationSeconds: body.EstimatedDurationSeconds, Blocks: workoutBlocks(body.Blocks),
	})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("workout"))
	}
	return c.Status(fiber.StatusCreated).JSON(response.WorkoutFromEntity(result))
}

func (h *V1) GetWorkout(c fiber.Ctx) error {
	const function = "GetWorkout"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	workoutID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	result, err := h.services.Workouts.GetByID(c.Context(), actorID, workoutID)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("workout"))
	}
	return c.JSON(response.WorkoutFromEntity(result))
}

func (h *V1) UpdateWorkout(c fiber.Ctx) error {
	const function = "UpdateWorkout"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	workoutID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	var body request.UpdateWorkout
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	result, err := h.services.Workouts.Update(c.Context(), usecase.UpdateWorkoutCommand{
		ActorID: actorID, WorkoutID: workoutID, Title: body.Title, Description: body.Description,
		SportType: body.SportType, ScheduledAt: body.ScheduledAt, EstimatedDurationSeconds: body.EstimatedDurationSeconds,
		Blocks: workoutBlocks(body.Blocks), ExpectedVersion: body.ExpectedVersion,
	})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("workout"))
	}
	return c.JSON(response.WorkoutFromEntity(result))
}

func (h *V1) CancelWorkout(c fiber.Ctx) error {
	const function = "CancelWorkout"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	workoutID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	var body request.ExpectedVersion
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	if err = h.services.Workouts.Cancel(c.Context(), usecase.ChangeWorkoutStateCommand{ActorID: actorID, WorkoutID: workoutID, ExpectedVersion: body.ExpectedVersion}); err != nil {
		return h.usecaseError(c, function, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *V1) DeleteWorkout(c fiber.Ctx) error {
	const function = "DeleteWorkout"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	workoutID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	version, err := h.ifMatch(c, function)
	if err != nil {
		return err
	}
	if err = h.services.Workouts.Delete(c.Context(), usecase.ChangeWorkoutStateCommand{ActorID: actorID, WorkoutID: workoutID, ExpectedVersion: version}); err != nil {
		return h.usecaseError(c, function, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *V1) ListAthleteWorkouts(c fiber.Ctx) error {
	const function = "ListAthleteWorkouts"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	athleteID, err := h.pathUUID(c, function, "athleteID")
	if err != nil {
		return err
	}
	from, to, err := h.timeRange(c, function)
	if err != nil {
		return err
	}
	result, err := h.services.Workouts.ListByAthlete(c.Context(), actorID, athleteID, from, to)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	return c.JSON(response.WorkoutsFromEntities(result))
}

func (h *V1) ListCoachWorkouts(c fiber.Ctx) error {
	const function = "ListCoachWorkouts"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	coachID, err := h.pathUUID(c, function, "coachID")
	if err != nil {
		return err
	}
	if coachID != actorID {
		return h.fail(c, function, "validate coach identity", entities.ErrForbidden, fiber.StatusForbidden, "forbidden", "operation is forbidden")
	}
	from, to, err := h.timeRange(c, function)
	if err != nil {
		return err
	}
	result, err := h.services.Workouts.ListByCoach(c.Context(), actorID, from, to)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	return c.JSON(response.WorkoutsFromEntities(result))
}

func workoutBlocks(values []request.WorkoutBlock) []usecase.WorkoutBlockCommand {
	result := make([]usecase.WorkoutBlockCommand, 0, len(values))
	for _, block := range values {
		exercises := make([]usecase.WorkoutExerciseCommand, 0, len(block.Exercises))
		for _, exercise := range block.Exercises {
			exercises = append(exercises, usecase.WorkoutExerciseCommand{
				ID: exercise.ID, Name: exercise.Name, Type: exercise.Type, Position: exercise.Position, Target: exercise.Target,
			})
		}
		result = append(result, usecase.WorkoutBlockCommand{
			ID: block.ID, Name: block.Name, Type: entities.WorkoutBlockType(block.Type), Position: block.Position, Exercises: exercises,
		})
	}
	return result
}

func requireDifferentIDs(first, second uuid.UUID) error {
	if first == second {
		return errors.New("identifiers must differ")
	}
	return nil
}
