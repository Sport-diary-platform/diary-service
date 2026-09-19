# diary-service

Микросервис спортивного дневника: связь тренера со спортсменом, тренировочные планы,
тренировки и результаты, ежедневные показатели, цели и комментарии. Сервис хранит данные
в PostgreSQL, публикует события через Kafka и применяет миграции при запуске.

## Запуск

1. Создайте общую сеть платформы: `docker network create common_net` (если её ещё нет).
2. При необходимости скопируйте `.env.example` в `.env` и задайте адреса auth/profile-service.
3. Запустите сервис: `docker compose up --build`.

При запуске через Compose API доступен на `http://localhost:8082`. Проверка состояния:
`GET /health/live` (без авторизации).

## HTTP API

Все маршруты ниже имеют префикс `/api/v1` и требуют заголовок
`Authorization: Bearer <JWT>`. В JWT обязательны UUID в `sub`, роль `user` или `admin`,
корректные `iss`, `aud` и `exp`. Тела запросов передаются как `application/json`.

| Раздел | Маршруты |
|---|---|
| Связи | `POST /relationships`; `POST /relationships/:id/terminate`; `GET /relationships/athletes`; `GET /relationships/coaches` |
| Планы | `POST /training-plans`; `GET, PUT /training-plans/:id`; `POST /training-plans/:id/{activate,complete,cancel}` |
| Тренировки | `POST /workouts`; `GET, PUT, DELETE /workouts/:id`; `POST /workouts/:id/cancel`; `GET /athletes/:athleteID/workouts`; `GET /coaches/:coachID/workouts` |
| Результаты | `POST, GET, PUT /workouts/:id/result` |
| Самочувствие | `POST /check-ins`; `GET /check-ins`; `GET, PUT /check-ins/:date` |
| Цели | `POST /goals`; `GET /goals`; `GET, PUT /goals/:id`; `POST /goals/:id/{complete,cancel}` |
| Комментарии | `POST, GET /workouts/:workoutID/comments`; `PUT /workouts/:workoutID/comments/:commentID` |

### Обязательные данные

- UUID в параметрах пути должен быть валидным. Даты имеют формат `YYYY-MM-DD`, время — RFC 3339.
- Связь: `athlete_id` (создаёт тренер). Завершение: `expected_version`.
- План: `athlete_id`, `name`, `start_date`; обновление также требует `expected_version`.
- Тренировка: `athlete_id`, `title`, `sport_type`, `scheduled_at`; обновление также требует `expected_version`. Блок содержит `name`, `type` (`warmup`, `main`, `cooldown`, `other`), а упражнение — `name`, `type`, `target`.
- Результат: `performed_at`, `rpe` и `feeling` (1–10), `expected_workout_version`; при обновлении вместо него нужен `expected_version`.
- Check-in: при создании обязателен `date`; показатели самочувствия имеют диапазон 1–10. Для обновления нужен `expected_version`.
- Цель: `title`, `type` (`performance`, `weight`, `strength`, `distance`, `time`, `custom`) и JSON-поле `target`; для обновления нужен `expected_version`.
- Комментарий: `text`; для обновления также `expected_version`.
- Операции смены состояния принимают `{ "expected_version": 1 }`. Удаление тренировки вместо тела требует заголовок `If-Match: 1`.
- Списки тренировок требуют query-параметры `from` и `to` в RFC 3339; список check-in — те же параметры в формате `YYYY-MM-DD`.

Полная модель и бизнес-правила описаны в [`diary-service-design.md`](diary-service-design.md).

## Проверки

```bash
go test ./...
go vet ./...
go test -tags=integration ./tests/integration
```

Integration-suite находится отдельно в `tests/integration` и запускает PostgreSQL через
testcontainers, поэтому для него нужен доступный Docker daemon.
