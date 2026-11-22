# Тихий план

Минималистичный MVP планнера задач. Backend построен на Go + Gin и SQLite, frontend — на React + Vite.

## Запуск

В первом терминале:

```sh
cd backend
go mod tidy
go run .
```

Перед первым запуском создайте `backend/.env` на основе `.env.example` и задайте уникальный `AUTH_SECRET` длиной не менее 32 символов. Приложение использует его для подписи access и refresh JWT. Не публикуйте `.env` и не добавляйте его в Git.

Сервер API доступен на `http://localhost:8080`, база `planner.db` создаётся автоматически. Во втором терминале:

```sh
cd frontend
npm install
npm run dev
```

## Авторизация и API

Frontend сначала отправляет логин-почту на `POST /api/user/lookup`. Backend сообщает, есть ли такой аккаунт; затем frontend вызывает `POST /api/user/login` или `POST /api/user/register`. Для обновления сессии используется `POST /api/user/refresh`, для выхода — `POST /api/user/logout`.

Маршруты задач (`GET`, `POST`, `PATCH`, `DELETE /api/tasks`) требуют `Authorization: Bearer <accessToken>`. Access token действует 15 минут, refresh token — 30 дней. Каждая задача хранит `user_id` и возвращается только её владельцу.

Пароль должен содержать не менее 8 символов и хотя бы одну цифру. В SQLite сохраняется bcrypt-хеш пароля; `PasswordHash` исключён из JSON-ответов.

