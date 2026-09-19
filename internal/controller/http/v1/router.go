package v1

import "github.com/gofiber/fiber/v3"

func NewRoutes(apiV1Group fiber.Router, handler *V1) {
	apiV1Group.Post("/training-plans", handler.CreateTrainingPlan)
	apiV1Group.Get("/training-plans/:id", handler.GetTrainingPlan)
	apiV1Group.Put("/training-plans/:id", handler.UpdateTrainingPlan)
	apiV1Group.Post("/training-plans/:id/activate", handler.ActivateTrainingPlan)
	apiV1Group.Post("/training-plans/:id/complete", handler.CompleteTrainingPlan)
	apiV1Group.Post("/training-plans/:id/cancel", handler.CancelTrainingPlan)

	apiV1Group.Post("/workouts", handler.CreateWorkout)
	apiV1Group.Get("/workouts/:id", handler.GetWorkout)
	apiV1Group.Put("/workouts/:id", handler.UpdateWorkout)
	apiV1Group.Post("/workouts/:id/cancel", handler.CancelWorkout)
	apiV1Group.Delete("/workouts/:id", handler.DeleteWorkout)
	apiV1Group.Get("/athletes/:athleteID/workouts", handler.ListAthleteWorkouts)
	apiV1Group.Get("/coaches/:coachID/workouts", handler.ListCoachWorkouts)

	apiV1Group.Post("/workouts/:id/result", handler.CreateWorkoutResult)
	apiV1Group.Get("/workouts/:id/result", handler.GetWorkoutResult)
	apiV1Group.Put("/workouts/:id/result", handler.UpdateWorkoutResult)

	apiV1Group.Post("/relationships", handler.CreateRelationship)
	apiV1Group.Post("/relationships/:id/terminate", handler.TerminateRelationship)
	apiV1Group.Get("/relationships/athletes", handler.ListRelationshipAthletes)
	apiV1Group.Get("/relationships/coaches", handler.ListRelationshipCoaches)

	apiV1Group.Post("/check-ins", handler.CreateCheckIn)
	apiV1Group.Get("/check-ins", handler.ListCheckIns)
	apiV1Group.Get("/check-ins/:date", handler.GetCheckIn)
	apiV1Group.Put("/check-ins/:date", handler.UpdateCheckIn)

	apiV1Group.Post("/goals", handler.CreateGoal)
	apiV1Group.Get("/goals", handler.ListGoals)
	apiV1Group.Get("/goals/:id", handler.GetGoal)
	apiV1Group.Put("/goals/:id", handler.UpdateGoal)
	apiV1Group.Post("/goals/:id/complete", handler.CompleteGoal)
	apiV1Group.Post("/goals/:id/cancel", handler.CancelGoal)

	apiV1Group.Post("/workouts/:workoutID/comments", handler.CreateComment)
	apiV1Group.Get("/workouts/:workoutID/comments", handler.ListComments)
	apiV1Group.Put("/workouts/:workoutID/comments/:commentID", handler.UpdateComment)
}
