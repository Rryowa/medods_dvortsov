Postman коллекция https://www.postman.com/rryowa/workspace/medods-asdvortsov/collection/27242165-76b440dd-ddaa-48a1-abb5-1e9bc8123513?action=share&creator=27242165&active-environment=27242165-a497a243-c116-4c4a-b09b-bf4e3d99d884

Base URL: `http://localhost:8080/api/v1`

пример .env:

JWT_SECRET и AUTH_SERVICE_API_KEY можно сгенерировать с помощью `make keygen`
Также рандомный GUID генерируется при старте сервиса

```shell
JWT_SECRET=XXX
AUTH_SERVICE_API_KEY=XXX

REDIS_ADDR=localhost:6379

POSTGRES_USER=admin
POSTGRES_PASSWORD=secret
POSTGRES_DB=mydb
SSL=disable
DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5432/${POSTGRES_DB}?sslmode=${SSL}

WEBHOOK_URL=http://localhost:9090
```

Как я понял Service-to-service, потому что в требованиях нет OIDC ( response_type=id_token … , openid-scope, nonce и тд)  
тогда как для браузера/мобилки чаще применяют PKCE + OIDC

По идее сервис работает с API Gateway - единая точка входа

1.  Клиент делает запрос с access-токеном в API Gateway
2.  Gateway перехватывает запрос, проверяет токен и проксирует запрос к нужному микросервису
3.  Для операций с токенами (выпуск, обновление) Gateway это доверенный клиент -
    он добавляет свой `X-API-Key` и перенаправляет запрос в этот сервис. (Реализована zero-downtime замена API ключа)

## Архитектура

```mermaid
graph TD
    subgraph "User/API Gateway"
        direction LR
        User[Client]
        Gateway(API Gateway)
    end

    subgraph "Presentation Layer (API)"
        A[Middleware]
        B[Controller]
    end

    subgraph "Business Logic Layer (Service)"
        C[AuthService]
        D[APIKeyService]
        E[TokenService]
        W[WebhookService]
    end

    subgraph "Data Access Layer (Storage)"
        F["storage.Storage\n(Interface)"]
        F_PG["Postgres Storage\n(Users & Sessions)"]
        F_RD["Redis TokenStorage\n(Denylist)"]
    end

    subgraph "Infrastructure"
        I[PostgresDB]
        J[Redis]
        K[External Webhook]
    end

    %% Dependencies
    User -- "Access/Refresh Tokens" --> Gateway
    Gateway -- "X-API-Key or Bearer" --> A

    A -- "Rate Limit" --> J
    A -- "Auth (API Key)" --> D
    A -- "Auth (Bearer)" --> C
    A -- "Routes to" --> B

    B -- "Uses" --> C

    C -- "Creates/Validates Tokens" --> E
    C -- "Manages Users/Sessions" --> F
    C -- "Notifies on IP change" --> W

    D -- "Stores/Validates API Keys" --> J
    E -- "Manages Denylist" --> F_RD
    W -- "Sends POST" --> K

    F -- "Implemented by" --> F_PG
    F_PG -- "CRUD" --> I
    F_RD -- "SET/GET" --> J
```

- **API Layer**: прием HTTP, валидация, передача в слой бизнес-логики
- **Service Layer**: Бизнес-логикаа - Не зависит от деталей бд
- **Storage Layer**: Интерфейсы для работы с данными + **транзакционная логика**
- **Infrastructure**: Токены и сессии - PostgreSQL, Кэш, denylist, RateLimiter - Redis

## Описание Сервисов (Service Layer)

Слой бизнес-логики инкапсулирует основные операции системы.

### `AuthService`

**Оркестратор аутентификации.** Связывает все части вместе:

- **`IssueTokens`**: Управляет процессом выпуска новой пары токенов.
- **`RefreshTokens`**: Реализует логику ротации и защиты от кражи refresh-токенов.
- **`Logout`**: Обеспечивает полный выход из системы (отзыв access и refresh токенов).
- **`AuthenticateAccessToken`**: Проверяет access-токен для middleware.

### `TokenService`

**Фабрика токенов.** Отвечает за криптографию и жизненный цикл токенов.

- **Создание**: Генерирует подписанные JWT (access) и компоненты `selector`/`verifier` (refresh).
- **Валидация**: Проверяет подписи, сроки жизни и **черный список (denylist)** для access-токенов.
- **Отзыв**: Помещает access-токены в denylist в Redis.

### `APIKeyService`

**Управление ключом API.** Обеспечивает безопасную S2S (service-to-service) аутентификацию.

- **`IsValidAPIKey`**: Проверяет предоставленный `X-API-Key`.
- **`SyncAPIKey`**: Реализует **бесшовную ротацию** ключа при перезапуске сервиса, используя "льготный период" для старого ключа.

Base URL: `/api/v1`

### Выдача новой пары токенов

- **Endpoint**: `POST /auth/tokens`
- **Описание**: Создает и возвращает новую пару `access` и `refresh` токенов для пользователя. Если пользователь с указанным `guid` не найден, будет создан новый.
- **Аутентификация**: Требует `X-API-Key` в заголовке. Этот ключ должен быть известен только доверенным клиентам (например, API Gateway)

### Обновление пары токенов

- **Endpoint**: `POST /auth/tokens/refresh`
- **Описание**: Обменивает валидный `refresh_token` на новую пару `access` и `refresh` токенов. Реализует механизм **Token Rotation**.
- **Аутентификация**: Требует `access_token` (может быть просроченным) в заголовке `Authorization` и `refresh_token` в `http-only` cookie.
- **Ответы**:
  - `200 OK`: Успешное обновление.
  - `401 Unauthorized`: Если refresh-токен невалиден, просрочен или отсутствует.

### Деавторизация пользователя (Logout)

- **Endpoint**: `POST /auth/logout`
- **Описание**: Немедленно отзывает текущий `access_token` (добавляя его в denylist) и удаляет все `refresh_token` сессии пользователя из базы данных.
- **Аутентификация**: Требует валидный `access_token`.
- **Ответы**:
  - `204 No Content`: Успешный выход.
  - `401 Unauthorized`: Если access-токен невалиден.

### Получение GUID пользователя

- **Endpoint**: `GET /auth/user/guid`
- **Описание**: Возвращает публичный `guid` пользователя, которому принадлежит `access_token`.
- **Аутентификация**: Требует валидный `access_token`.
- **Ответы**:
  - `200 OK`: Успешное получение GUID.
  - `401 Unauthorized`: Если access-токен невалиден, просрочен или отозван.

## Middleware

1.  **Логирование**
2.  **Rate Limiter**: Ограничивает число запросов с одного IP для защиты от брутфорса.
    - **Алгоритм**: Sliding window, реализован на Redis (Lua) для атомарности.
    - **Настройки**: `RATE_LIMIT_LIMIT`, `RATE_LIMIT_INTERVAL`, `RATE_LIMIT_BLOCK_TIME`.
    - **Ответ**: `429 Too Many Requests` с заголовком `Retry-After`.
3.  **Валидация OpenAPI и Аутентификация**: проверяет каждый запрос на соответствие спецификации `openapi.yaml` + выполняет аутентификацию, вызывая кастомный `Authenticator`, который проверяет либо `X-API-Key`, либо `Bearer` access-токен

## БД

- **`users`**:

  - `id (BIGSERIAL)`: Внутренний, автоинкрементный ID
  - `guid (UUID)`: Внешний, публичный идентификатор пользователя

- **`sessions`**:
  - `user_id`: Внешний ключ к `users.id`
  - `selector (TEXT)`: Уникальный селектор для поиска сессии
  - `verifier_hash (TEXT)`: Хеш верификации токена
  - `status (TEXT)`: Статус сессии

#### Оптимизация (Индексы)

- `CREATE INDEX ON sessions (selector)`

  - Поиск сессии по refresh-токену

- `CREATE INDEX ON sessions (user_id)`
  - Ускоряет удаление всех сессий пользователя

## Безопасность

### 1. Безопасное хранение Refresh Токенов (Selector/Verifier)

- **Формат токена**: `часть_для_поиска.секретная_часть` (`selector.verifier`)
- **Selector**: Публичная, случайная строка. Используется как "логин" для поска сессии в базе данных (`WHERE selector = ?`) - хранится в БД
- **Verifier**: Секретная, случайная строка. Клиент отправляет ее на сервер. Сервер **хэширует** полученный `verifier` и сравнивает его с **хэшем** - `verifier_hash`

### 2. Ротация Refresh Токенов

- При каждом обновлении пары токенов (`/auth/tokens/refresh`) старый refresh-токен помечается как использованный, и клиенту выдается **новая пара** access и refresh токенов

- При попытке использовать "использованный" refresh-токен - **все активные сессии юзера отзываются**

### 3. Отзыв Access Токенов (Server-Side Logout)

- При вызове `POST /auth/logout` Idшник access-токена помещается в **черный список (denylist)** в Redis
- Любая попытка использовать токен будет отклонена middleware, даже если его срок жизни (`exp`) еще не истек

### 4. Ротация API-ключа (X-API-Key)

Замена API-ключа производится без остановки (zero-downtime)

**Процесс замены:**

1.  Обновить переменную `AUTH_SERVICE_API_KEY`
2.  Перезапустить сервис
3.  Механизм ротации (`SyncAPIKey`):
    - При старте сервис сравнивает хэш нового ключа из переменной окружения с хэшем текущего ключа в Redis (`apikey:current`)
    - Если они не совпадают, хеш старого ключа перемещается в `apikey:old` с TTL (24 часа)
    - Хэш нового ключа становится `apikey:current`
4.  В течение 24 часов система будет принимать запросы как со старым, так и с новым API-ключом.

### 5. Webhook-уведомления

- **Событие**: Смена IP-адреса при обновлении токена (`/auth/tokens/refresh`).
- **Действие**: Отправляет `POST` запрос на `WEBHOOK_URL`.
- **Payload**: `user_id`, `old_ip`, `new_ip`, `user_agent`.
- **Настройка**: Переменная окружения `WEBHOOK_URL`.
