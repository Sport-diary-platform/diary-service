package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

const (
	defaultEnvironment  = "local"
	defaultHTTPPort     = 8080
	defaultJWTIssuer    = "sport-diary-auth"
	defaultJWTAudience  = "sport-diary-api"
	defaultJWTClockSkew = 30 * time.Second
	defaultLogLevel     = "info"
)

type Config struct {
	Environment string
	HTTP        HTTP
	Database    Database
	Auth        Auth
	LogLevel    Logger
	Kafka       Kafka
}

type HTTP struct {
	Port int `env:"HTTP_PORT,required"`
}

type Database struct {
	URL string `env:"DATABASE_URL,required,unset"`
}

type Auth struct {
	JWKSURL   string        `env:"AUTH_JWKS_URL,required,unset"`
	Issuer    string        `env:"JWT_ISSUER,required,unset"`
	Audience  string        `env:"JWT_AUDIENCE,required,unset"`
	ClockSkew time.Duration `env:"JWT_CLOCK_SKEW,required,unset"`
}

type Logger struct {
	Level string `env:"LOG_LEVEL,required"`
}

type Kafka struct {
	Seeds             []string `env:"KAFKA_SEEDS,unset"`
	TopicCoachAthlete string   `env:"KAFKA_TOPIC_COACHATHLETE"`
	TopicTrainingPlan string   `env:"KAFKA_TOPIC_TRAININGPLAN"`
	TopicWorkout      string   `env:"KAFKA_TOPIC_WORKOUT"`
	TopiWorkoutResult string   `env:"KAFKA_TOPIC_WORKOUTRESULT"`
	TopicDailyCheckIn string   `env:"KAFKA_TOPIC_DAILYCHECKIN"`
	TopicGoal         string   `env:"KAFKA_TOPIC_GOAL"`
}

func NewConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}
	return cfg, nil
}
