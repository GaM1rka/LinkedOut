# LinkedOut

LinkedOut — Telegram-бот для студентов и junior IT-специалистов. За короткое интервью бот помогает превратить сырой рассказ о pet, учебном, хакатонном или рабочем проекте в честное и конкретное описание для резюме.

## Архитектура

- `bot-service` — Telegram-бот на Go: ведёт интервью, вызывает LLM, показывает результат, собирает feedback, отправляет события и данные в metrics-service.
- `metrics-service` — REST API на Go + PostgreSQL: хранит пользователей, интервью, ответы, генерации, feedback, ручные проверки и считает MVP-метрики.
- `postgres` — база данных для metrics-service.

RAG, GitHub/HH/LinkedIn-интеграции, оплата и веб-фронтенд в MVP не реализуются.

## Конфигурация

Создайте локальный `.env` из примера:

```bash
cp .env.example .env
```

Заполните в `.env` реальные значения:

```env
BOT_TOKEN=
BOT_ADMIN_IDS=
LLM_API_KEY=
LLM_BASE_URL=
LLM_MODEL=skald-loki
```

Секреты не хранятся в репозитории.

## Запуск

```bash
docker compose up --build
```

Сервисы:

- metrics-service: `http://localhost:8080`
- PostgreSQL: `localhost:5432`

## Swagger

Swagger UI доступен после запуска:

```text
http://localhost:8080/swagger/index.html
```

Если нужно перегенерировать Swagger-аннотации через swag CLI:

```bash
swag init -g cmd/metrics-service/main.go -o docs
```

## Команды бота

- `/start` — показать описание и кнопку начала интервью.
- `/cancel` — сбросить текущее in-memory интервью.
- `/stats` — админская статистика MVP.
- `/pending_reviews` — последние 5 генераций без ручной проверки.
- `/review <generation_id> <status> <comment>` — сохранить ручную проверку.

Статусы ручной проверки:

```text
pass
fail_water
fail_inaccuracy
fail_exaggeration
fail_other
```

Пример:

```text
/review 123e4567-e89b-12d3-a456-426614174000 pass Хорошее описание
```

## Пользовательский flow

1. Пользователь запускает `/start` и нажимает «Начать интервью».
2. Выбирает целевую роль: Backend, Frontend, QA, Analyst, Data/ML или Другое.
3. Выбирает тип проекта: Pet-project, учебный проект, хакатон, стартап, рабочий/стажировка или Другое.
4. Отвечает максимум на 7 вопросов.
5. Бот вызывает LLM и генерирует:
   - 3-5 bullet points;
   - краткое описание проекта;
   - подтверждённые навыки;
   - предупреждения о воде, неточностях и накрутке опыта.
6. Бот собирает feedback и сохраняет его в metrics-service.

## MVP-метрика

`GET /api/v1/metrics/summary` возвращает:

- всего начатых интервью;
- всего завершённых интервью;
- completion rate;
- среднее время интервью;
- долю пользователей, готовых вставить описание в резюме;
- число пользователей, готовых платить;
- pass rate ручной проверки;
- `mvp_success`;
- `not_enough_data`.

`mvp_success = true`, если одновременно:

- среднее время интервью не больше 15 минут;
- `ready_to_use_rate >= 0.7`;
- `payment_willing_users >= 3`;
- `manual_review_pass_rate >= 0.7`.

Если недостаточно данных, например нет feedback или manual reviews, `not_enough_data = true`, а `mvp_success = false`.

## Тесты

```bash
go test ./...
```
