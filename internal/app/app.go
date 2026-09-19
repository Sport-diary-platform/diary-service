package app

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"diary-service/config"
	profileclient "diary-service/internal/client/profile"
	httpcontroller "diary-service/internal/controller/http"
	"diary-service/internal/controller/http/middleware"
	v1 "diary-service/internal/controller/http/v1"
	"diary-service/internal/outbox"
	repositorypostgres "diary-service/internal/repository/postgres"
	"diary-service/internal/usecase"
	"diary-service/pkg/httpserver"
	"diary-service/pkg/kafka/producer"
	"diary-service/pkg/logger"
	postgresclient "diary-service/pkg/postgres"
)

func Run(cfg config.Config) error {
	log := logger.New(cfg.LogLevel.Level)
	if err := Migrate(cfg.Database.URL); err != nil {
		return errors.Join(errors.New("app - Run - Migrate"), err)
	}

	database, err := postgresclient.New(cfg.Database.URL, postgresclient.MaxPoolSize(20))
	if err != nil {
		return errors.Join(errors.New("app - Run - postgres.New"), err)
	}
	defer database.Close()

	profiles, err := profileclient.New(cfg.Profile.URL, cfg.Profile.Timeout)
	if err != nil {
		return errors.Join(errors.New("app - Run - profile.New"), err)
	}

	relationships := repositorypostgres.NewRelationshipRepository(database.Pool)
	plans := repositorypostgres.NewTrainingPlanRepository(database.Pool)
	workouts := repositorypostgres.NewWorkoutRepository(database.Pool)
	results := repositorypostgres.NewWorkoutResultRepository(database.Pool)
	checkIns := repositorypostgres.NewDailyCheckInRepository(database.Pool)
	goals := repositorypostgres.NewGoalRepository(database.Pool)
	comments := repositorypostgres.NewCommentRepository(database.Pool)
	outboxRepository := repositorypostgres.NewOutboxRepository(database.Pool)
	transactions := repositorypostgres.NewTransactionManager(database.Pool)

	services := &v1.Services{
		Relationships:  usecase.NewRelationshipUseCase(relationships, outboxRepository, transactions, profiles),
		TrainingPlans:  usecase.NewTrainingPlanUseCase(plans, relationships, outboxRepository, transactions, profiles),
		Workouts:       usecase.NewWorkoutUseCase(workouts, results, plans, relationships, outboxRepository, transactions, profiles),
		WorkoutResults: usecase.NewWorkoutResultUseCase(results, workouts, relationships, outboxRepository, transactions, profiles),
		CheckIns:       usecase.NewDailyCheckInUseCase(checkIns, outboxRepository, transactions, profiles),
		Goals:          usecase.NewGoalUseCase(goals, relationships, outboxRepository, transactions, profiles),
		Comments:       usecase.NewCommentUseCase(comments, workouts, relationships, transactions, profiles),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	auth, err := middleware.NewJWT(ctx, middleware.JWTConfig{
		JWKSURL: cfg.Auth.JWKSURL, Issuer: cfg.Auth.Issuer,
		Audience: cfg.Auth.Audience, ClockSkew: cfg.Auth.ClockSkew,
	}, log)
	if err != nil {
		return errors.Join(errors.New("app - Run - middleware.NewJWT"), err)
	}

	server := httpserver.New(httpserver.Port(strconv.Itoa(cfg.HTTP.Port)))
	httpcontroller.NewRouter(server.App, v1.New(services, log), auth.Handler)
	server.Start()

	kafkaProducer, err := producer.New(cfg.Kafka.Seeds, producer.SnappyCompression())
	if err != nil {
		_ = server.Shutdown()
		return errors.Join(errors.New("app - Run - producer.New"), err)
	}
	defer kafkaProducer.Close()
	pingCtx, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	err = kafkaProducer.Ping(pingCtx)
	cancelPing()
	if err != nil {
		_ = server.Shutdown()
		return errors.Join(errors.New("app - Run - producer.Ping"), err)
	}

	worker := outbox.NewWorker(outboxRepository, kafkaProducer, log, outbox.Topics{
		CoachAthlete: cfg.Kafka.TopicCoachAthlete, TrainingPlan: cfg.Kafka.TopicTrainingPlan,
		Workout: cfg.Kafka.TopicWorkout, WorkoutResult: cfg.Kafka.TopicWorkoutResult,
		DailyCheckIn: cfg.Kafka.TopicDailyCheckIn, Goal: cfg.Kafka.TopicGoal,
	}, cfg.Kafka.OutboxPollInterval, cfg.Kafka.OutboxLease, cfg.Kafka.OutboxBatchSize)
	go worker.Run(ctx)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.ShutdownContext(shutdownCtx); err != nil {
			return errors.Join(errors.New("app - Run - ShutdownContext"), err)
		}
		return nil
	case err := <-server.Notify():
		stop()
		if err == nil {
			return nil
		}
		return fmt.Errorf("app - Run - HTTP server: %w", err)
	}
}
