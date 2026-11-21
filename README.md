# Тихий план

Минималистичный MVP планнера задач. Backend построен на Go + Gin и SQLite, frontend — на React + Vite.

## Запуск

В первом терминале:

```sh
cd backend
go mod tidy
go run .
```

Сервер API доступен на `http://localhost:8080`, база `planner.db` создаётся автоматически. Во втором терминале:

```sh
cd frontend
npm install
npm run dev
```

Откройте адрес Vite (обычно `http://localhost:5173`). Для production сборки используйте `npm run build`.

## API

- `GET /api/tasks` — список задач
- `POST /api/tasks` — создать задачу (`title`, `description`, `startAt` в ISO 8601)
- `PATCH /api/tasks/:id` — отметить задачу (`done`)
- `DELETE /api/tasks/:id` — удалить задачу
