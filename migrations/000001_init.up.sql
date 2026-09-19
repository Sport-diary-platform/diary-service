CREATE TABLE IF NOT EXISTS coach_athlete_relationships (
    id UUID PRIMARY KEY,
    coach_id UUID NOT NULL,
    athlete_id UUID NOT NULL,
    status VARCHAR(32) NOT NULL,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT chk_relationship_different_users
        CHECK (coach_id <> athlete_id),

    CONSTRAINT chk_relationship_status
        CHECK (status IN ('active', 'terminated')),

    CONSTRAINT chk_relationship_version
        CHECK (version > 0)
);

CREATE UNIQUE INDEX uq_active_relationship
    ON coach_athlete_relationships(coach_id, athlete_id)
    WHERE status = 'active';

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
    version BIGINT NOT NULL DEFAULT 1,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT chk_training_plan_dates
        CHECK (end_date IS NULL OR end_date >= start_date),

    CONSTRAINT chk_training_plan_status
        CHECK (status IN ('draft', 'active', 'completed', 'cancelled')),

    CONSTRAINT chk_training_plan_version
        CHECK (version > 0)
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
    estimated_duration_seconds INTEGER,

    status VARCHAR(32) NOT NULL,
    version BIGINT NOT NULL DEFAULT 1,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT chk_workout_duration
        CHECK (
            estimated_duration_seconds IS NULL
            OR estimated_duration_seconds > 0
        ),

    CONSTRAINT chk_workout_status
        CHECK (status IN ('planned', 'completed', 'cancelled')),

    CONSTRAINT chk_workout_version
        CHECK (version > 0),

    CONSTRAINT fk_workout_training_plan FOREIGN KEY (training_plan_id)
        REFERENCES training_plans(id) ON DELETE SET NULL
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
        UNIQUE (workout_id, position),

    CONSTRAINT chk_workout_block_position
        CHECK (position >= 0),

    CONSTRAINT fk_workout_block_workout FOREIGN KEY (workout_id)
        REFERENCES workouts(id) ON DELETE CASCADE
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
        UNIQUE (workout_block_id, position),

    CONSTRAINT chk_exercise_position
        CHECK (position >= 0),

    CONSTRAINT fk_exercise_workout_block FOREIGN KEY (workout_block_id)
        REFERENCES workout_blocks(id) ON DELETE CASCADE
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
    version BIGINT NOT NULL DEFAULT 1,

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
        ),

    CONSTRAINT chk_workout_result_version
        CHECK (version > 0),

    CONSTRAINT fk_workout_result_workout FOREIGN KEY (workout_id)
        REFERENCES workouts(id) ON DELETE RESTRICT
);

CREATE INDEX idx_workout_results_athlete
    ON workout_results(athlete_id, performed_at);



CREATE TABLE exercise_results (
    id UUID PRIMARY KEY,

    workout_result_id UUID NOT NULL,
    exercise_id UUID NOT NULL,

    actual JSONB NOT NULL,

    comment TEXT,
	version BIGINT NOT NULL DEFAULT 1,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT uq_exercise_result
        UNIQUE (workout_result_id, exercise_id),
	CONSTRAINT chk_exercise_result_version
		CHECK (version > 0),
    CONSTRAINT fk_exercise_result_workout_result FOREIGN KEY (workout_result_id)
        REFERENCES workout_results(id) ON DELETE CASCADE,

    CONSTRAINT fk_exercise_result_exercise FOREIGN KEY (exercise_id)
        REFERENCES exercises(id) ON DELETE RESTRICT
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
    version BIGINT NOT NULL DEFAULT 1,

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
        CHECK (motivation BETWEEN 1 AND 10),

    CONSTRAINT chk_daily_check_in_version
        CHECK (version > 0)
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
    version BIGINT NOT NULL DEFAULT 1,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT chk_goal_status
        CHECK (status IN ('active', 'completed', 'cancelled')),

    CONSTRAINT chk_goal_type
        CHECK (type IN ('performance', 'weight', 'distance', 'time', 'strength', 'custom')),

    CONSTRAINT chk_goal_version
        CHECK (version > 0)
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
    version BIGINT NOT NULL DEFAULT 1,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT chk_workout_comment_version
        CHECK (version > 0),

    CONSTRAINT fk_workout_comment_workout FOREIGN KEY (workout_id)
        REFERENCES workouts(id) ON DELETE CASCADE
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
    published_at TIMESTAMPTZ,
    available_at TIMESTAMPTZ NOT NULL,
    locked_until TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,

    CONSTRAINT chk_outbox_event_version
        CHECK (event_version > 0),

    CONSTRAINT chk_outbox_attempts
        CHECK (attempts >= 0)
);

CREATE INDEX idx_outbox_unpublished
    ON outbox_events(available_at, created_at)
    WHERE published_at IS NULL;
