CREATE TABLE IF N0T EXISTS coach_athlete_relationships (
    id UUID PRIMARY KEY,
    coach_id UUID NOT NULL,
    athlete_id UUID NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT chk_relationship_different_users
        CHECK (coach_id <> athlete_id),

    CONSTRAINT uq_relationship
        UNIQUE (coach_id, athlete_id)
);

CREATE INDEX idx_relationships_coach
    ON coach_athlete_relationships(coach_id);

CREATE INDEX idx_relationships_athlete
    ON coach_athlete_relationships(athlete_id);

CREATE TABLE training_plans (
    id UUID PRIMARY KEY,
    coach_id UUID NOT NULL,
    athlete_id UUID NOT NULL,

    name VARCHAR(255) NOT NULL,
    description TEXT,

    start_date DATE NOT NULL,
    end_date DATE,

    status VARCHAR(32) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT chk_training_plan_dates
        CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_training_plans_athlete
    ON training_plans(athlete_id, start_date);

CREATE INDEX idx_training_plans_coach
    ON training_plans(coach_id);


CREATE TABLE workouts (
    id UUID PRIMARY KEY,

    training_plan_id UUID,

    coach_id UUID NOT NULL,
    athlete_id UUID NOT NULL,

    title VARCHAR(255) NOT NULL,
    description TEXT,

    sport_type VARCHAR(64) NOT NULL,

    scheduled_at TIMESTAMPTZ NOT NULL,
    estimated_duration INTEGER,

    status VARCHAR(32) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT chk_workout_duration
        CHECK (
            estimated_duration IS NULL
            OR estimated_duration > 0
        )
    CONSTRAINT fk_training_plan_id FOREIGN KEY training_plan_id 
        REFERENCES training_plans(id)
);


CREATE INDEX idx_workouts_athlete_schedule
    ON workouts(athlete_id, scheduled_at);

CREATE INDEX idx_workouts_coach_schedule
    ON workouts(coach_id, scheduled_at);

CREATE INDEX idx_workouts_plan
    ON workouts(training_plan_id);



CREATE TABLE workout_blocks (
    id UUID PRIMARY KEY,

    workout_id UUID NOT NULL,

    name VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL,
    position INTEGER NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT uq_workout_block_position
        UNIQUE (workout_id, position)
    CONSTRAINT fk_workout_id FOREIGN KEY workout_id 
    REFERENCES workouts(id)
);

CREATE TABLE exercises (
    id UUID PRIMARY KEY,

    workout_block_id UUID NOT NULL,

    name VARCHAR(255) NOT NULL,
    type VARCHAR(64) NOT NULL,
    position INTEGER NOT NULL,

    target JSONB NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT uq_exercise_position
        UNIQUE (workout_block_id, position)
    CONSTRAINT fk_workout_block_id FOREIGN KEY workout_block_id
    REFERENCES workout_blocks(id)
);


CREATE TABLE workout_results (
    id UUID PRIMARY KEY,

    workout_id UUID NOT NULL,
    athlete_id UUID NOT NULL,

    performed_at TIMESTAMPTZ NOT NULL,

    duration_seconds INTEGER,
    distance_meters NUMERIC(10, 2),

    avg_heart_rate SMALLINT,
    max_heart_rate SMALLINT,

    calories INTEGER,

    rpe SMALLINT NOT NULL,
    feeling SMALLINT NOT NULL,

    comment TEXT,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT uq_workout_result_workout
        UNIQUE (workout_id),

    CONSTRAINT chk_workout_result_rpe
        CHECK (rpe BETWEEN 1 AND 10),

    CONSTRAINT chk_workout_result_feeling
        CHECK (feeling BETWEEN 1 AND 10),

    CONSTRAINT chk_workout_result_duration
        CHECK (
            duration_seconds IS NULL
            OR duration_seconds > 0
        ),

    CONSTRAINT chk_workout_result_distance
        CHECK (
            distance_meters IS NULL
            OR distance_meters >= 0
        )
    CONSTRAINT fk_workout_id FOREIGN KEY workout_id
    REFERENCES workouts(id)
);

CREATE INDEX idx_workout_results_athlete
    ON workout_results(athlete_id, performed_at);



CREATE TABLE exercise_results (
    id UUID PRIMARY KEY,

    workout_result_id UUID NOT NULL,
    exercise_id UUID NOT NULL,

    actual JSONB NOT NULL,

    comment TEXT,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT uq_exercise_result
        UNIQUE (workout_result_id, exercise_id)
    CONSTRAINT fk_workout_result_id FOREIGN KEY workout_result_id
        REFERENCES workout_results(id)
    CONSTRAINT fk_exercise_id FOREIGN KEY exercise_id 
        REFERENCES exercises(id)
);




CREATE TABLE daily_check_ins (
    id UUID PRIMARY KEY,

    athlete_id UUID NOT NULL,
    date DATE NOT NULL,

    sleep_duration_min INTEGER,
    sleep_quality SMALLINT,

    energy SMALLINT,
    stress SMALLINT,
    soreness SMALLINT,
    motivation SMALLINT,

    weight_kg NUMERIC(6, 2),
    resting_heart_rate SMALLINT,

    comment TEXT,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT uq_daily_check_in
        UNIQUE (athlete_id, date),

    CONSTRAINT chk_sleep_quality
        CHECK (sleep_quality BETWEEN 1 AND 10),

    CONSTRAINT chk_energy
        CHECK (energy BETWEEN 1 AND 10),

    CONSTRAINT chk_stress
        CHECK (stress BETWEEN 1 AND 10),

    CONSTRAINT chk_soreness
        CHECK (soreness BETWEEN 1 AND 10),

    CONSTRAINT chk_motivation
        CHECK (motivation BETWEEN 1 AND 10)
);


CREATE TABLE goals (
    id UUID PRIMARY KEY,

    athlete_id UUID NOT NULL,
    coach_id UUID,

    title VARCHAR(255) NOT NULL,
    description TEXT,

    type VARCHAR(64) NOT NULL,
    target JSONB NOT NULL,

    deadline DATE,

    status VARCHAR(32) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_goals_athlete
    ON goals(athlete_id);

CREATE INDEX idx_goals_coach
    ON goals(coach_id);



CREATE TABLE workout_comments (
    id UUID PRIMARY KEY,

    workout_id UUID NOT NULL,
    author_id UUID NOT NULL,

    text TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL

    CONSTRAINT fk_workout_id FOREIGN KEY workout_id 
        REFERENCES workouts(id)
);

CREATE INDEX idx_workout_comments_workout
    ON workout_comments(workout_id, created_at);



CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,

    event_type VARCHAR(255) NOT NULL,
    event_version INTEGER NOT NULL,

    aggregate_type VARCHAR(64) NOT NULL,
    aggregate_id UUID NOT NULL,

    payload JSONB NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished
    ON outbox_events(created_at)
    WHERE published_at IS NULL;