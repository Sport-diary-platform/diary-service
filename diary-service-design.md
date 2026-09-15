# Diary Service — проектирование микросервиса спортивного дневника

## 1. Назначение

`diary-service` — микросервис спортивного дневника для платформы тренеров и спортсменов.

Основные задачи сервиса:

- тренер назначает спортсмену тренировки;
- тренер формирует тренировочные планы;
- спортсмен видит назначенные тренировки;
- спортсмен после тренировки вручную отправляет фактический результат;
- спортсмен ведёт ежедневный check-in;
- тренер и спортсмен работают с целями;
- тренер и спортсмен могут оставлять комментарии к тренировкам;
- сервис публикует доменные события через Kafka;
- сервис не отвечает за авторизацию пользователей;


Первая версия сознательно не содержит:
- автоматической фиксации начала тренировки;
- автоматической фиксации окончания тренировки;
- статуса `in_progress`;
- интеграций с GPS/часами/пульсометрами;
- email/social login;
- сторонних OAuth-провайдеров;
- competitions;
- полноценной библиотеки упражнений.

---

# 2. Стек

- Go
- Fiber
- PostgreSQL
- Kafka
- Docker
- JWT используется auth-service; diary-service получает идентификатор пользователя из JWT-контекста

---

# 3. Архитектура

Используется Clean Architecture с простой слоистой структурой:

```text
project/
├── go.mod
├── go.sum
│
├── Dockerfile
│
├── cmd/
│   └── main.go
│
├── internal/
│   ├── app/
│   │
│   ├── entities/
│   │
│   ├── usecase/
│   │
│   ├── repository/
│   │
│   └── controller/
│
├── pkg/
│   ├── logger/
│   │   └── logger.go
│   │
│   ├── postgres/
│   │    ├── options.go
│   │    └── postgres.go
│   │
│   └── httpserver/
│       ├── options.go
│       └── fiber.go
│
└── migrations/
```

### Ответственность слоёв

#### `internal/entities`

Domain entities, enums, domain errors и domain-методы.

Здесь нет:
- PostgreSQL;
- Fiber;
- Kafka;
- HTTP;
- JWT;
- repository interfaces;

Entities создаются/заполняются usecase/repository-кодом.

#### `internal/usecase`

Application/business logic.

Содержит:
- use case interfaces;
- команды/DTO usecase-уровня, если необходимы;
- реализацию use cases;
- orchestration между entities и repository;
- authorization/business checks;
- transaction boundaries.

#### `internal/repository`

Контракты repository layer и их реализации.

Здесь находятся repository interfaces в стиле:

```go
type WorkoutRepository interface {
    Create(ctx context.Context, workout *entities.Workout) error
    GetByID(ctx context.Context, workoutID uuid.UUID) (*entities.Workout, error)
    Update(ctx context.Context, workout *entities.Workout) error
}
```

и PostgreSQL implementations.

Важно: контракт repository layer находится непосредственно в `internal/repository`, согласно принятой структуре проекта.

#### `internal/handler`

Fiber HTTP handlers, request parsing, validation HTTP-входа, response mapping.


#### `internal/app`

Сборка приложения:
- repositories;
- usecases;
- handlers;
- router;
- dependencies;
- lifecycle;
- graceful shutdown.

#### `pkg`

Общие технические компоненты:
- logger;
- PostgreSQL connection;
- Fiber HTTP server.

---

# 4. Главный принцип доменной модели

Ключевое разделение:

```text
Workout      = назначенный план тренировки
WorkoutResult = фактическое выполнение тренировки
```

Пример:

```text
Workout

5 × 1000 m
target pace: 4:00/km
rest: 2 min
```

После тренировки спортсмен создаёт:

```text
WorkoutResult

3:58
4:01
4:05
4:02
4:08

RPE: 8
Feeling: 6/10
Comment: "Последние два интервала были тяжелыми"
```

Наличие `WorkoutResult` означает, что по тренировке есть фактический результат.

---

# 5. Aggregates

Основные aggregate roots:

1. `CoachAthleteRelationship`
2. `TrainingPlan`
3. `Workout`
4. `WorkoutResult`
5. `DailyCheckIn`
6. `Goal`

Вспомогательные entities:

- `WorkoutBlock`
- `Exercise`
- `ExerciseResult`
- `WorkoutComment`

---

# 6. CoachAthleteRelationship

Связывает тренера и спортсмена.

```text
CoachAthleteRelationship
├── ID
├── CoachID
├── AthleteID
├── Status
├── CreatedAt
└── UpdatedAt
```

Статусы:

```text
active
terminated
```

Инварианты:

- `coach_id != athlete_id`;
- одна связь между конкретным coach и athlete;
- тренер может работать с athlete только при активной связи.

База:

```sql
UNIQUE (coach_id, athlete_id)
CHECK (coach_id <> athlete_id)
```

`coach_id` и `athlete_id` являются UUID пользователей из auth-service.

Foreign key на auth-service отсутствует, поскольку базы микросервисов независимы.

---

# 7. TrainingPlan

Тренировочный план спортсмена.

```text
TrainingPlan
├── ID
├── CoachID
├── AthleteID
├── Name
├── Description
├── StartDate
├── EndDate
├── Status
├── CreatedAt
└── UpdatedAt
```

Статусы:

```text
draft
active
completed
cancelled
```

Допустимые переходы:

```text
draft -> active
draft -> cancelled

active -> completed
active -> cancelled
```

Недопустимые переходы:

```text
completed -> active
completed -> cancelled

cancelled -> active
cancelled -> completed
```

`EndDate` может быть `NULL`.

Если `EndDate` задан:

```text
EndDate >= StartDate
```

---

# 8. Workout

Назначенная тренером тренировка.

```text
Workout
├── ID
├── TrainingPlanID *
├── CoachID
├── AthleteID
├── Title
├── Description
├── SportType
├── ScheduledAt
├── EstimatedDuration
├── Status
├── Blocks[]
├── CreatedAt
└── UpdatedAt
```

`TrainingPlanID` nullable: тренировка может быть разовой и не относиться к плану.

Статусы:

```text
planned
cancelled
```

Никаких:

```text
in_progress
skipped
```

### Инварианты

- title не пустой;
- coach и athlete заданы;
- scheduled time задан;
- estimated duration, если задан, > 0;
- block position уникален внутри workout;
- exercise position уникален внутри block.

### Domain methods

Предполагаемые методы:

```go
func (w *Workout) Cancel() error
func (w *Workout) AddBlock(block WorkoutBlock) error
func (w *Workout) RemoveBlock(blockID uuid.UUID) error
```

Методы не должны содержать инфраструктурную логику.

---

# 9. WorkoutBlock

Составная часть Workout.

Примеры:

```text
Warm-up
Main
Cool-down
```

Структура:

```text
WorkoutBlock
├── ID
├── WorkoutID
├── Name
├── Type
├── Position
├── Exercises[]
├── CreatedAt
└── UpdatedAt
```

Типы:

```text
warmup
main
cooldown
other
```

Constraint:

```sql
UNIQUE (workout_id, position)
```

---

# 10. Exercise

Упражнение относится к конкретному WorkoutBlock.

```text
Exercise
├── ID
├── WorkoutBlockID
├── Name
├── Type
├── Position
├── Target
├── CreatedAt
└── UpdatedAt
```

`Target` хранится как JSONB.

Примеры.

Running:

```json
{
  "distance_meters": 1000,
  "repetitions": 5,
  "target_pace_seconds": 240,
  "rest_seconds": 120
}
```

Strength:

```json
{
  "sets": 4,
  "repetitions": 8,
  "weight_kg": 100
}
```

Swimming:

```json
{
  "distance_meters": 100,
  "repetitions": 10,
  "style": "freestyle",
  "rest_seconds": 30
}
```

Constraint:

```sql
UNIQUE (workout_block_id, position)
```

---

# 11. WorkoutResult

Фактическое выполнение Workout, отправленное спортсменом после тренировки.

```text
WorkoutResult
├── ID
├── WorkoutID
├── AthleteID
├── PerformedAt
├── DurationSeconds
├── DistanceMeters
├── AvgHeartRate
├── MaxHeartRate
├── Calories
├── RPE
├── Feeling
├── Comment
├── ExerciseResults[]
├── CreatedAt
└── UpdatedAt
```

Связь:

```text
Workout 1 ---- 0..1 WorkoutResult
```

В БД:

```sql
UNIQUE (workout_id)
```

### Почему только один result

После тренировки спортсмен отправляет один итоговый отчёт.

При необходимости он редактируется через update.

Удаление результата в первой версии не предусмотрено.

### RPE

Шкала 1–10.

```sql
CHECK (rpe BETWEEN 1 AND 10)
```

### Feeling

Общее субъективное состояние 1–10.

```sql
CHECK (feeling BETWEEN 1 AND 10)
```

RPE и feeling — разные показатели.

### Другие поля

`duration_seconds > 0`, если задан.

`distance_meters >= 0`, если задан.

Heart rate и calories не следует чрезмерно ограничивать DB CHECK'ами; это относится к application/domain validation.

---

# 12. ExerciseResult

Фактический результат конкретного упражнения.

```text
ExerciseResult
├── ID
├── WorkoutResultID
├── ExerciseID
├── Actual
├── Comment
├── CreatedAt
└── UpdatedAt
```

`Actual` — JSONB.

Пример:

```json
{
  "sets": [
    {
      "reps": 8,
      "weight": 100
    },
    {
      "reps": 8,
      "weight": 100
    },
    {
      "reps": 8,
      "weight": 95
    },
    {
      "reps": 6,
      "weight": 95
    }
  ]
}
```

Constraint:

```sql
UNIQUE (workout_result_id, exercise_id)
```

Application/domain invariant:

```text
exercise.workout_id == workout_result.workout_id
```

То есть ExerciseResult нельзя привязать к exercise из другой тренировки.

---

# 13. DailyCheckIn

Ежедневная оценка состояния спортсмена.

```text
DailyCheckIn
├── ID
├── AthleteID
├── Date
├── SleepDurationMin
├── SleepQuality
├── Energy
├── Stress
├── Soreness
├── Motivation
├── WeightKg
├── RestingHeartRate
├── Comment
├── CreatedAt
└── UpdatedAt
```

Главный invariant:

```text
один athlete + одна дата = один DailyCheckIn
```

База:

```sql
UNIQUE (athlete_id, date)
```

Оценки:

```sql
CHECK (sleep_quality BETWEEN 1 AND 10)
CHECK (energy BETWEEN 1 AND 10)
CHECK (stress BETWEEN 1 AND 10)
CHECK (soreness BETWEEN 1 AND 10)
CHECK (motivation BETWEEN 1 AND 10)
```

---

# 14. Goal

Цель спортсмена.

```text
Goal
├── ID
├── AthleteID
├── CoachID *
├── Title
├── Description
├── Type
├── Target
├── Deadline
├── Status
├── CreatedAt
└── UpdatedAt
```

`CoachID` nullable: спортсмен может создать личную цель.

Типы:

```text
performance
weight
distance
time
strength
custom
```

`Target` — JSONB.

Пример:

```json
{
  "type": "time",
  "distance": "10km",
  "target_seconds": 2400
}
```

Статусы:

```text
active
completed
cancelled
```

---

# 15. WorkoutComment

Комментарии тренера и спортсмена к тренировке.

```text
WorkoutComment
├── ID
├── WorkoutID
├── AuthorID
├── Text
├── CreatedAt
└── UpdatedAt
```

`AuthorID` — UUID пользователя из auth-service.

Authorization:

- coach может комментировать тренировку своего athlete;
- athlete может комментировать собственную тренировку.

---

# 16. OutboxEvent

Transactional outbox для публикации событий в Kafka.

```text
OutboxEvent
├── ID
├── EventType
├── EventVersion
├── AggregateType
├── AggregateID
├── Payload
├── CreatedAt
└── PublishedAt
```

`Payload` — JSONB.

`PublishedAt = NULL` означает, что событие ещё не опубликовано.

Индекс:

```sql
CREATE INDEX idx_outbox_unpublished
ON outbox_events(created_at)
WHERE published_at IS NULL;
```

---

# 17. PostgreSQL схема

## coach_athlete_relationships

```sql
CREATE TABLE coach_athlete_relationships (
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
```

Индексы:

```sql
CREATE INDEX idx_relationships_coach
    ON coach_athlete_relationships(coach_id);

CREATE INDEX idx_relationships_athlete
    ON coach_athlete_relationships(athlete_id);
```

---

## training_plans

```sql
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
```

Индексы:

```sql
CREATE INDEX idx_training_plans_athlete
    ON training_plans(athlete_id, start_date);

CREATE INDEX idx_training_plans_coach
    ON training_plans(coach_id);
```

---

## workouts

```sql
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
);
```

Индексы:

```sql
CREATE INDEX idx_workouts_athlete_schedule
    ON workouts(athlete_id, scheduled_at);

CREATE INDEX idx_workouts_coach_schedule
    ON workouts(coach_id, scheduled_at);

CREATE INDEX idx_workouts_plan
    ON workouts(training_plan_id);
```

---

## workout_blocks

```sql
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
);
```

Индекс `UNIQUE` покрывает поиск по `(workout_id, position)`.

---

## exercises

```sql
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
);
```

---

## workout_results

```sql
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
);
```

Индекс:

```sql
CREATE INDEX idx_workout_results_athlete
    ON workout_results(athlete_id, performed_at);
```

---

## exercise_results

```sql
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
);
```

---

## daily_check_ins

```sql
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
```

---

## goals

```sql
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
```

Индексы:

```sql
CREATE INDEX idx_goals_athlete
    ON goals(athlete_id);

CREATE INDEX idx_goals_coach
    ON goals(coach_id);
```

---

## workout_comments

```sql
CREATE TABLE workout_comments (
    id UUID PRIMARY KEY,

    workout_id UUID NOT NULL,
    author_id UUID NOT NULL,

    text TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
```

Индекс:

```sql
CREATE INDEX idx_workout_comments_workout
    ON workout_comments(workout_id, created_at);
```

---

## outbox_events

```sql
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
```

Индекс:

```sql
CREATE INDEX idx_outbox_unpublished
    ON outbox_events(created_at)
    WHERE published_at IS NULL;
```

---

# 18. Foreign keys

Внутри diary-service логические связи должны контролироваться БД.

Рекомендуемые FK:

```text
workout_blocks.workout_id -> workouts.id

exercises.workout_block_id -> workout_blocks.id

workout_results.workout_id -> workouts.id

exercise_results.workout_result_id -> workout_results.id

exercise_results.exercise_id -> exercises.id

workout_comments.workout_id -> workouts.id
```

Для relationships/training plans/workouts с `coach_id`/`athlete_id` FK на auth-service не используется.

Для `training_plan_id` -> `training_plans.id` рекомендуется FK.

При удалении parent-сущности следует использовать поведение, соответствующее lifecycle. В большинстве случаев бизнес-операции должны использовать domain status/soft lifecycle, а не физическое удаление.

---

# 19. API

Базовый prefix:

```text
/api/v1
```

## Training plans

```text
POST   /training-plans
GET    /training-plans/:id
PUT    /training-plans/:id
POST   /training-plans/:id/activate
POST   /training-plans/:id/complete
POST   /training-plans/:id/cancel
```

## Workouts

```text
POST   /workouts
GET    /workouts/:id
PUT    /workouts/:id
POST   /workouts/:id/cancel
```

Списки:

```text
GET /athletes/:athleteID/workouts
GET /coaches/:coachID/workouts
```

Фактическое выполнение:

```text
POST /workouts/:id/result
GET  /workouts/:id/result
PUT  /workouts/:id/result
```

---

# 20. Create Workout Result

Главный endpoint первой версии:

```http
POST /api/v1/workouts/:id/result
```

Пример request:

```json
{
  "performed_at": "2026-09-05T18:30:00Z",
  "duration_seconds": 3200,
  "distance_meters": 8400,
  "avg_heart_rate": 158,
  "max_heart_rate": 177,
  "calories": 620,
  "rpe": 8,
  "feeling": 6,
  "comment": "Последние два интервала были тяжелыми",
  "exercises": [
    {
      "exercise_id": "uuid",
      "actual": {
        "intervals": [
          {
            "distance_meters": 1000,
            "duration_seconds": 238
          }
        ]
      },
      "comment": ""
    }
  ]
}
```

Usecase должен:

1. получить Workout;
2. проверить, что текущий пользователь — athlete этого Workout;
3. проверить, что Workout не cancelled;
4. проверить, что WorkoutResult ещё не существует;
5. проверить ExerciseResult → Exercise принадлежность этому Workout;
6. создать WorkoutResult;
7. создать ExerciseResults;
8. создать OutboxEvent;
9. выполнить всё в одной транзакции.

---

# 21. Update Workout Result

```http
PUT /api/v1/workouts/:id/result
```

Изменяет существующий результат.

В первой версии:

```text
DELETE result
```

не поддерживается.

Причина: тренировочная история является важными данными.

---

# 22. Что означает пропущенная тренировка

Нет статуса:

```text
skipped
```

Если:

```text
Workout exists
AND WorkoutResult does not exist
AND scheduled_at < now
```

сервис аналитики/UI может интерпретировать это как пропущенную/невыполненную тренировку.

Таким образом `skipped` является вычисляемым состоянием, а не состоянием доменной сущности.

---

# 23. Частично выполненная тренировка

Если Workout:

```text
5 × 1000m
```

а спортсмен выполнил:

```text
3 × 1000m
```

WorkoutResult просто содержит фактические данные для трёх интервалов.

Не нужно вводить:

```text
partially_completed
```

Completion percentage можно вычислять аналитически.

---

# 24. Workout analysis

В будущем можно иметь компонент:

```text
WorkoutResultAnalyzer
```

который получает:

```text
planned Workout
+
actual WorkoutResult
```

и рассчитывает:

```text
completion_percent
deviation
training_load
```

Пример:

```json
{
  "completion_percent": 87,
  "training_load": 480,
  "rpe": 8
}
```

В первой версии результаты анализа не обязательно сохранять в БД.

Сначала метрики должны быть определены на практике.

---

# 25. Authorization

Diary-service не реализует login/password.

Auth-service отвечает за:

```text
login
password
JWT generation
JWT validation
```

Diary-service получает:

```text
userID
role
```

из JWT middleware/context.

Роли:

```text
user
admin
```

Для получения типа профиля (athlete/coach) следует сделать GET запрос к profile-service:

```
curl http://localhost:8081/api/v1/internal/profiles/{user_id} 
```

response:
```
{"profile_type":"athlete"}
```

### Примеры

Athlete может:

```text
read own workouts
create own workout results
update own workout results
create/update own check-ins
read own goals
create own goals
```

Coach может:

```text
create workouts for connected athletes
update workouts
cancel workouts
read workout results of connected athletes
create/update training plans
manage goals according to permissions
comment on workouts
```

Критически важная проверка:

```text
JWT userID == workout.athleteID
```

для создания результата.

Тренер не может создавать WorkoutResult от имени спортсмена.

---

# 26. Repository contracts

Контракты repository layer находятся в:

```text
internal/repository
```

Пример.

## WorkoutRepository

```go
type WorkoutRepository interface {
    Create(ctx context.Context, workout *entities.Workout) error
    GetByID(ctx context.Context, workoutID uuid.UUID) (*entities.Workout, error)
    Update(ctx context.Context, workout *entities.Workout) error
    Delete(ctx context.Context, workoutID uuid.UUID) error

    ListByAthlete(
        ctx context.Context,
        athleteID uuid.UUID,
        from, to time.Time,
    ) ([]entities.Workout, error)

    ListByCoach(
        ctx context.Context,
        coachID uuid.UUID,
        from, to time.Time,
    ) ([]entities.Workout, error)
}
```

Фактический набор методов может быть скорректирован под use cases.

---

# 27. TrainingPlanRepository

```go
type TrainingPlanRepository interface {
    Create(ctx context.Context, plan *entities.TrainingPlan) error
    GetByID(ctx context.Context, planID uuid.UUID) (*entities.TrainingPlan, error)
    Update(ctx context.Context, plan *entities.TrainingPlan) error

    ListByAthlete(
        ctx context.Context,
        athleteID uuid.UUID,
    ) ([]entities.TrainingPlan, error)

    ListByCoach(
        ctx context.Context,
        coachID uuid.UUID,
    ) ([]entities.TrainingPlan, error)
}
```

---

# 28. WorkoutResultRepository

```go
type WorkoutResultRepository interface {
    Create(
        ctx context.Context,
        result *entities.WorkoutResult,
    ) error

    GetByWorkoutID(
        ctx context.Context,
        workoutID uuid.UUID,
    ) (*entities.WorkoutResult, error)

    Update(
        ctx context.Context,
        result *entities.WorkoutResult,
    ) error

    ExistsByWorkoutID(
        ctx context.Context,
        workoutID uuid.UUID,
    ) (bool, error)
}
```

---

# 29. DailyCheckInRepository

```go
type DailyCheckInRepository interface {
    Create(
        ctx context.Context,
        checkIn *entities.DailyCheckIn,
    ) error

    GetByDate(
        ctx context.Context,
        athleteID uuid.UUID,
        date time.Time,
    ) (*entities.DailyCheckIn, error)

    Update(
        ctx context.Context,
        checkIn *entities.DailyCheckIn,
    ) error

    List(
        ctx context.Context,
        athleteID uuid.UUID,
        from, to time.Time,
    ) ([]entities.DailyCheckIn, error)
}
```

---

# 30. GoalRepository

```go
type GoalRepository interface {
    Create(ctx context.Context, goal *entities.Goal) error
    GetByID(ctx context.Context, goalID uuid.UUID) (*entities.Goal, error)
    Update(ctx context.Context, goal *entities.Goal) error

    ListByAthlete(
        ctx context.Context,
        athleteID uuid.UUID,
    ) ([]entities.Goal, error)
}
```

---

# 31. RelationshipRepository

```go
type RelationshipRepository interface {
    Create(
        ctx context.Context,
        relationship *entities.CoachAthleteRelationship,
    ) error

    Get(
        ctx context.Context,
        coachID, athleteID uuid.UUID,
    ) (*entities.CoachAthleteRelationship, error)

    IsActive(
        ctx context.Context,
        coachID, athleteID uuid.UUID,
    ) (bool, error)

    Terminate(
        ctx context.Context,
        relationshipID uuid.UUID,
    ) error

    ListAthletes(
        ctx context.Context,
        coachID uuid.UUID,
    ) ([]uuid.UUID, error)

    ListCoaches(
        ctx context.Context,
        athleteID uuid.UUID,
    ) ([]uuid.UUID, error)
}
```

---

# 32. CommentRepository

```go
type CommentRepository interface {
    Create(
        ctx context.Context,
        comment *entities.WorkoutComment,
    ) error

    ListByWorkout(
        ctx context.Context,
        workoutID uuid.UUID,
    ) ([]entities.WorkoutComment, error)

    Update(
        ctx context.Context,
        comment *entities.WorkoutComment,
    ) error
}
```

---

# 33. OutboxRepository

```go
type OutboxRepository interface {
    Create(
        ctx context.Context,
        event *entities.OutboxEvent,
    ) error

    GetUnpublished(
        ctx context.Context,
        limit int,
    ) ([]entities.OutboxEvent, error)

    MarkPublished(
        ctx context.Context,
        eventID uuid.UUID,
    ) error
}
```

---

# 34. TransactionManager

Для операций, которые изменяют несколько таблиц и outbox, нужен transaction boundary.

```go
type TransactionManager interface {
    WithTransaction(
        ctx context.Context,
        fn func(ctx context.Context) error,
    ) error
}
```

Для CreateWorkoutResult транзакция должна включать:

```text
workout_results
exercise_results
outbox_events
```

А commit должен происходить только после успешной записи всех данных.

---

# 35. Usecase contracts

Интерфейсы usecase находятся в:

```text
internal/usecase
```

## WorkoutUseCase

```go
type WorkoutUseCase interface {
    Create(
        ctx context.Context,
        cmd CreateWorkoutCommand,
    ) (*entities.Workout, error)

    GetByID(
        ctx context.Context,
        workoutID uuid.UUID,
    ) (*entities.Workout, error)

    Update(
        ctx context.Context,
        cmd UpdateWorkoutCommand,
    ) (*entities.Workout, error)

    Cancel(
        ctx context.Context,
        workoutID uuid.UUID,
    ) error

    ListByAthlete(
        ctx context.Context,
        athleteID uuid.UUID,
        from, to time.Time,
    ) ([]entities.Workout, error)

    ListByCoach(
        ctx context.Context,
        coachID uuid.UUID,
        from, to time.Time,
    ) ([]entities.Workout, error)
}
```

---

# 36. WorkoutResultUseCase

```go
type WorkoutResultUseCase interface {
    Create(
        ctx context.Context,
        cmd CreateWorkoutResultCommand,
    ) (*entities.WorkoutResult, error)

    GetByWorkoutID(
        ctx context.Context,
        workoutID uuid.UUID,
    ) (*entities.WorkoutResult, error)

    Update(
        ctx context.Context,
        cmd UpdateWorkoutResultCommand,
    ) (*entities.WorkoutResult, error)
}
```

---

# 37. TrainingPlanUseCase

```go
type TrainingPlanUseCase interface {
    Create(
        ctx context.Context,
        cmd CreateTrainingPlanCommand,
    ) (*entities.TrainingPlan, error)

    GetByID(
        ctx context.Context,
        planID uuid.UUID,
    ) (*entities.TrainingPlan, error)

    Update(
        ctx context.Context,
        cmd UpdateTrainingPlanCommand,
    ) (*entities.TrainingPlan, error)

    Activate(ctx context.Context, planID uuid.UUID) error
    Complete(ctx context.Context, planID uuid.UUID) error
    Cancel(ctx context.Context, planID uuid.UUID) error
}
```

---

# 38. DailyCheckInUseCase

```go
type DailyCheckInUseCase interface {
    Create(
        ctx context.Context,
        cmd CreateDailyCheckInCommand,
    ) (*entities.DailyCheckIn, error)

    GetByDate(
        ctx context.Context,
        athleteID uuid.UUID,
        date time.Time,
    ) (*entities.DailyCheckIn, error)

    Update(
        ctx context.Context,
        cmd UpdateDailyCheckInCommand,
    ) (*entities.DailyCheckIn, error)

    List(
        ctx context.Context,
        athleteID uuid.UUID,
        from, to time.Time,
    ) ([]entities.DailyCheckIn, error)
}
```

---

# 39. GoalUseCase

```go
type GoalUseCase interface {
    Create(
        ctx context.Context,
        cmd CreateGoalCommand,
    ) (*entities.Goal, error)

    GetByID(
        ctx context.Context,
        goalID uuid.UUID,
    ) (*entities.Goal, error)

    Update(
        ctx context.Context,
        cmd UpdateGoalCommand,
    ) (*entities.Goal, error)

    Complete(ctx context.Context, goalID uuid.UUID) error
    Cancel(ctx context.Context, goalID uuid.UUID) error
}
```

---

# 40. RelationshipUseCase

```go
type RelationshipUseCase interface {
    Create(
        ctx context.Context,
        cmd CreateRelationshipCommand,
    ) (*entities.CoachAthleteRelationship, error)

    Terminate(
        ctx context.Context,
        relationshipID uuid.UUID,
    ) error

    ListAthletes(
        ctx context.Context,
        coachID uuid.UUID,
    ) ([]uuid.UUID, error)

    ListCoaches(
        ctx context.Context,
        athleteID uuid.UUID,
    ) ([]uuid.UUID, error)
}
```

---

# 41. CommentUseCase

```go
type CommentUseCase interface {
    Create(
        ctx context.Context,
        cmd CreateCommentCommand,
    ) (*entities.WorkoutComment, error)

    ListByWorkout(
        ctx context.Context,
        workoutID uuid.UUID,
    ) ([]entities.WorkoutComment, error)

    Update(
        ctx context.Context,
        cmd UpdateCommentCommand,
    ) (*entities.WorkoutComment, error)
}
```

---

# 42. Domain errors

В `internal/entities` должны быть определены ошибки, отражающие domain/application semantics.

Примеры:

```go
var (
    ErrWorkoutNotFound              = errors.New("workout not found")
    ErrWorkoutCancelled             = errors.New("workout is cancelled")
    ErrWorkoutResultAlreadyExists   = errors.New("workout result already exists")
    ErrWorkoutResultNotFound        = errors.New("workout result not found")

    ErrInvalidRPE                   = errors.New("invalid rpe")
    ErrInvalidFeeling               = errors.New("invalid feeling")

    ErrTrainingPlanNotFound         = errors.New("training plan not found")
    ErrInvalidTrainingPlanState     = errors.New("invalid training plan state")

    ErrRelationshipNotFound         = errors.New("coach athlete relationship not found")
    ErrRelationshipNotActive        = errors.New("coach athlete relationship is not active")

    ErrDailyCheckInAlreadyExists    = errors.New("daily check-in already exists")
    ErrDailyCheckInNotFound         = errors.New("daily check-in not found")

    ErrGoalNotFound                 = errors.New("goal not found")
)
```

Repository-specific errors могут находиться в `internal/repository`, если это ошибки storage contract.

---

# 43. Business logic: CreateWorkout

Алгоритм:

```text
1. Authenticate user.
2. Verify user is coach.
3. Verify coach-athlete relationship is active.
4. Validate command.
5. Create Workout entity.
6. Save Workout.
7. Save blocks and exercises.
8. Create outbox event.
9. Commit transaction.
```

Event:

```text
workout.created
```

---

# 44. Business logic: CreateWorkoutResult

Алгоритм:

```text
1. Authenticate user.
2. Verify user is athlete.
3. Load Workout.
4. Verify workout.athlete_id == currentUserID.
5. Verify Workout is not cancelled.
6. Verify WorkoutResult does not exist.
7. Validate RPE and feeling.
8. Validate exercise IDs belong to Workout.
9. Create WorkoutResult.
10. Create ExerciseResults.
11. Create OutboxEvent.
12. Commit transaction.
```

Event:

```text
workout_result.created
```

---

# 45. Business logic: UpdateWorkoutResult

```text
1. Authenticate user.
2. Load result.
3. Verify current user is result athlete.
4. Validate new data.
5. Update result.
6. Update exercise results.
7. Create outbox event.
8. Commit transaction.
```

Event:

```text
workout_result.updated
```

---

# 46. Kafka events

Возможный набор:

```text
coach_athlete.created
coach_athlete.terminated

training_plan.created
training_plan.activated
training_plan.completed

workout.created
workout.updated
workout.cancelled

workout_result.created
workout_result.updated

daily_checkin.created
daily_checkin.updated

goal.created
goal.completed
```

---

# 47. Event envelope

Рекомендуемый формат:

```json
{
  "event_id": "uuid",
  "event_type": "workout_result.created",
  "event_version": 1,
  "occurred_at": "2026-09-05T18:35:00Z",
  "aggregate_type": "workout_result",
  "aggregate_id": "uuid",
  "payload": {
    "workout_id": "uuid",
    "athlete_id": "uuid",
    "coach_id": "uuid"
  }
}
```

`event_id` нужен для idempotency consumers.

---

# 48. Transactional Outbox

Нельзя делать:

```text
INSERT workout_result
COMMIT

Kafka.Publish()
```

потому что между DB commit и Kafka publish возможен crash.

Правильная схема:

```text
BEGIN

INSERT workout_result
INSERT exercise_results
INSERT outbox_event

COMMIT

Outbox worker
    ↓
Kafka
```

Если процесс упал после Kafka publish, но до `published_at`, событие может быть опубликовано повторно.

Поэтому consumers должны быть idempotent.

---


# 49. Entity relationships

Итоговая модель:

```text
CoachAthleteRelationship
          │
          ├───────────────┐
          │               │
          ▼               ▼
   TrainingPlan         Goal
          │
          │ 1:N
          ▼
       Workout
          │
     ┌────┴─────┐
     │          │
     ▼          ▼
WorkoutBlock  WorkoutResult
     │          │
     ▼          ▼
 Exercise    ExerciseResult

Workout
   │
   └── WorkoutComment

Athlete
   │
   └── DailyCheckIn
```

---

# 50. Необходимые индексы

```text
coach_athlete_relationships:
    (coach_id)
    (athlete_id)
    UNIQUE (coach_id, athlete_id)

training_plans:
    (athlete_id, start_date)
    (coach_id)

workouts:
    (athlete_id, scheduled_at)
    (coach_id, scheduled_at)
    (training_plan_id)

workout_blocks:
    UNIQUE (workout_id, position)

exercises:
    UNIQUE (workout_block_id, position)

workout_results:
    UNIQUE (workout_id)
    (athlete_id, performed_at)

daily_check_ins:
    UNIQUE (athlete_id, date)

goals:
    (athlete_id)
    (coach_id)

workout_comments:
    (workout_id, created_at)

outbox_events:
    partial index on published_at IS NULL
```

---

# 51. Рекомендации по UUID

UUID генерируется application-side.

Например:

```go
id := uuid.New()
```

---

# 52. Timestamps

Все timestamps:

```sql
TIMESTAMPTZ NOT NULL
```

В Go:

```go
time.Time
```

Для календарных дат:

```sql
DATE
```

Для них не следует использовать `time.Time` с timezone semantics, если значение действительно является календарной датой.

---

# 53. Nullable fields

Optional numerical fields должны быть nullable.

Например в Go:

```go
*int
*float64
```

для:

```text
DurationSeconds
DistanceMeters
AvgHeartRate
MaxHeartRate
Calories
```

Это позволяет различать:

```text
0
```

и:

```text
не указано
```

---

# 54. Handler responsibilities

Handler должен:

1. извлечь authenticated user из Fiber context;
2. распарсить path/query/body;
3. выполнить HTTP-level validation;
4. вызвать usecase;
5. преобразовать domain/application errors в HTTP status;
6. вернуть JSON response.

---

# 55. Usecase responsibilities

Usecase:

- orchestration;
- authorization checks;
- relationship checks;
- repository calls;
- transactions;
- создание domain entities;
- business rules;
- создание outbox events.

Usecase не должен знать:

```text
Fiber
HTTP status codes
SQL
Kafka implementation
```

---

# 56. Repository responsibilities

Repository:

- PostgreSQL queries;
- mapping DB rows ↔ entities;
- transaction implementation;
- locking при необходимости;
- persistence errors.

Repository не должен решать:

```text
может ли coach редактировать workout
может ли athlete создать result
можно ли завершить training plan
```

Это application/domain responsibility.

---

# 57. Concurrency

Особенно важно для:

```text
CreateWorkoutResult
```

Два параллельных запроса могут одновременно проверить:

```text result doesn't exist
```

Поэтому только application check недостаточен.

DB constraint:

```sql
UNIQUE (workout_id)
```

является последней защитой.

Usecase должен корректно преобразовать unique violation в:

```text
ErrWorkoutResultAlreadyExists
```

---

# 58. Concurrency для DailyCheckIn

Аналогично:

```sql
UNIQUE (athlete_id, date)
```

Даже если два запроса одновременно создают check-in, база гарантирует уникальность.

---

# 59. Список use cases первой версии

### Relationships

```text
CreateRelationship
TerminateRelationship
ListAthletes
ListCoaches
```

### Training plans

```text
CreateTrainingPlan
GetTrainingPlan
UpdateTrainingPlan
ActivateTrainingPlan
CompleteTrainingPlan
CancelTrainingPlan
```

### Workouts

```text
CreateWorkout
GetWorkout
UpdateWorkout
CancelWorkout
ListAthleteWorkouts
ListCoachWorkouts
```

### Workout results

```text
CreateWorkoutResult
GetWorkoutResult
UpdateWorkoutResult
```

### Daily check-ins

```text
CreateDailyCheckIn
GetDailyCheckIn
UpdateDailyCheckIn
ListDailyCheckIns
```

### Goals

```text
CreateGoal
GetGoal
UpdateGoal
CompleteGoal
CancelGoal
```

### Comments

```text
CreateComment
ListComments
UpdateComment
```

---

# 60. Итоговая архитектура

```text
                       ┌─────────────────┐
                       │   auth-service  │
                       │ login/password  │
                       │      JWT        │
                       └────────┬────────┘
                                │
                                │ JWT
                                ▼
┌─────────────────────────────────────────────────────┐
│                    diary-service                    │
│                                                     │
│  Fiber                                              │
│    │                                                │
│    ▼                                                │
│  handler                                            │
│    │                                                │
│    ▼                                                │
│  usecase                                            │
│    │                                                │
│    ▼                                                │
│  entities                                           │
│    ▲                                                │
│    │                                                │
│  repository                                         │
│    │                                                │
│    ▼                                                │
│  PostgreSQL                                         │
│                                                     │
│  Transaction:                                       │
│    domain data + outbox event                       │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
                  Kafka / Outbox
                       │
             ┌─────────┴──────────┐
             ▼                    ▼
     notification-service   analytics-service
             │
             ▼
         WebSocket
             │
             ▼
           Coach
```

---

# 61. Главные архитектурные инварианты

1. `Workout` — это назначение, а не lifecycle выполнения.
2. `WorkoutResult` — факт выполнения.
3. У Workout максимум один WorkoutResult.
4. WorkoutResult создаёт только athlete этого Workout.
5. `skipped` не является состоянием Workout.
6. Частичное выполнение выражается фактическими данными результата.
7. `DailyCheckIn` уникален для athlete + date.
8. Все изменения domain state выполняются через usecase.
9. Repository отвечает только за persistence.
10. Handler не содержит бизнес-логики.
11. Outbox event записывается в одной transaction с domain data.
12. Kafka delivery считается at-least-once.
13. Kafka consumers должны быть idempotent.
14. Diary-service не знает о WebSocket.
15. Diary-service не владеет пользователями; users принадлежат auth-service.
16. Между БД микросервисов нет foreign keys.
17. Entity не требует constructor.
18. Handler interface не создаётся без реального потребителя.
19. Repository contracts находятся в `internal/repository`.
20. Usecase contracts находятся в `internal/usecase`.
