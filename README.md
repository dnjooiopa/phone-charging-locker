# Phone Charging Locker

A backend service for managing phone charging lockers. Users select a locker, pay via QR code, and the locker unlocks for charging until the time expires.

## Architecture

This project follows **Clean Architecture** with clear separation of concerns:

```
cmd/pcl/              → Entry point, dependency injection
internal/
  domain/             → Entity structs (Locker, Session)
  usecase/            → Business logic, repository interfaces
  repository/         → SQLite implementations
  server/gin_server/  → HTTP handlers, routes, middleware
schema/               → Database migrations (embedded SQL)
pkg/dbctx/            → Database context helpers
pkg/tu/               → Test utilities
tests/                → Integration tests
```

## Tech Stack

- **Language:** Go
- **HTTP Framework:** Gin
- **Database:** SQLite (modernc.org/sqlite, pure Go)
- **QR Code:** skip2/go-qrcode
- **Testing:** testify

## Flow

1. User selects a locker
2. Backend generates a QR code for payment
3. User pays (manual confirmation endpoint - no 3rd party payment integration)
4. Locker status changes to charging
5. After time expires, locker is released automatically

## API Endpoints

| Method | Path                              | Description                  |
|--------|-----------------------------------|------------------------------|
| GET    | `/healthz`                        | Health check                 |
| POST   | `/lockers`                        | Create a new locker          |
| GET    | `/lockers`                        | List all lockers with status |
| DELETE | `/lockers/:id`                    | Delete locker and sessions   |
| POST   | `/lockers/:id/select`             | Select locker, get QR code   |
| GET    | `/sessions/:id`                   | Check session status         |
| POST   | `/sessions/:id/confirm-payment`   | Confirm payment (stub)       |

### Examples

**Create a locker**
```bash
curl -s -X POST http://localhost:8080/lockers \
  -H "Content-Type: application/json" \
  -d '{"name": "L-01"}'
```
```json
{
  "id": 1,
  "name": "L-01",
  "status": "available"
}
```

**List all lockers**
```bash
curl -s http://localhost:8080/lockers
```
```json
{
  "lockers": [
    { "id": 1, "name": "L-01", "status": "available" },
    { "id": 2, "name": "L-02", "status": "in_use" },
    { "id": 3, "name": "L-03", "status": "maintenance" }
  ]
}
```

**Delete a locker**
```bash
curl -s -X DELETE http://localhost:8080/lockers/1
```
Returns `204 No Content` on success. Deletes the locker and all related sessions.

**Select a locker**
```bash
curl -s -X POST http://localhost:8080/lockers/1/select
```
```json
{
  "session_id": 1,
  "qr_code_data": "PCL-PAY-1",
  "qr_code_png": "<base64-encoded PNG>"
}
```

**Confirm payment**
```bash
curl -s -X POST http://localhost:8080/sessions/1/confirm-payment
```
```json
{
  "session_id": 1,
  "status": "charging",
  "started_at": "2026-03-15T10:00:00Z",
  "expired_at": "2026-03-15T11:00:00Z"
}
```

**Check session status**
```bash
curl -s http://localhost:8080/sessions/1
```
```json
{
  "id": 1,
  "locker_id": 1,
  "status": "charging",
  "qr_code_data": "PCL-PAY-1",
  "amount": 2000,
  "started_at": "2026-03-15T10:00:00Z",
  "expired_at": "2026-03-15T11:00:00Z"
}
```

## Error Responses

```json
{
  "error_code": "LOCKER_NOT_FOUND",
  "error_message": "locker not found"
}
```

| Error Code             | HTTP Status | Description               |
|------------------------|-------------|---------------------------|
| VALIDATION_ERROR       | 400         | Invalid request params    |
| LOCKER_NAME_ALREADY_EXISTS | 409     | Locker name already taken |
| LOCKER_NOT_FOUND       | 404         | Locker does not exist     |
| LOCKER_NOT_AVAILABLE   | 409         | Locker is in use/maint.   |
| SESSION_NOT_FOUND      | 404         | Session does not exist    |
| SESSION_ALREADY_PAID   | 409         | Payment already confirmed |
| SESSION_EXPIRED        | 410         | Session has expired       |
| INVALID_SESSION_STATE  | 409         | Invalid state transition  |
| INTERNAL_SERVER_ERROR  | 500         | Unexpected server error   |

## Setup

### Prerequisites

- Go 1.25+

### Environment Variables

| Variable           | Description                         | Default                |
|--------------------|-------------------------------------|------------------------|
| ENVIRONMENT        | Runtime environment                 | development            |
| HOST               | Server bind host                    | 0.0.0.0                |
| PORT               | Server bind port                    | 8080                   |
| DB_PATH            | SQLite database file path           | ./data/pcl.db          |
| CHARGING_DURATION  | Charging session duration           | 1h                     |
| CHARGING_AMOUNT    | Charging amount in satang           | 2000 (20 THB)          |

### Run

```bash
# Copy and configure environment
cp .env.example .env

# Run with live reload
make dev

# Or build and run
go build -o .build/pcl ./cmd/pcl
./.build/pcl
```

### Database

Schema migrations run automatically on startup. The SQLite database file is created automatically at the configured `DB_PATH`. The initial migration creates:
- `locker` table with status (available, in_use, maintenance)
- `session` table with status (pending_payment, charging, completed, expired)

### Docker

```bash
docker build -t phone-charging-locker .
docker run -p 8080:8080 --env-file .env phone-charging-locker
```

## Testing

```bash
# Unit tests
make test-unit

# Unit tests with coverage
make test-unit-cover

# Unit tests with HTML coverage report
make test-unit-coverage

# Schema migration test
make test-schema

# Integration tests
make test-integration
```

## Session Expiry

Sessions are expired through two mechanisms:

1. **Background worker**: Runs every minute, finds all charging sessions past their `expired_at` time, and marks them as completed while releasing the locker.
2. **On-demand check**: When a session is queried via `GET /sessions/:id`, if it's expired, it's automatically completed before returning.

## Database Schema

```sql
-- Locker statuses: available, in_use, maintenance
-- Session statuses: pending_payment, charging, completed, expired

locker (id, name, status, created_at, updated_at)
session (id, locker_id, status, qr_code_data, amount, started_at, expired_at, created_at, updated_at)
```

- `amount` stored in satang (smallest currency unit) to avoid floating point
- `started_at` and `expired_at` are NULL until payment is confirmed
