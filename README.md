# Webhook Delivery Service

A reliable, multi-tenant webhook delivery service built with Go and PostgreSQL. It accepts events through an HTTP API, stores them persistently, and delivers them to configured tenant webhook endpoints with retry handling, HMAC-SHA256 signatures, and delivery history.

## Features

- Accept webhook events through `POST /events`.
- Persist events and delivery attempts in PostgreSQL.
- Deliver events in creation order for each tenant.
- Process different tenants independently.
- Retry failed deliveries using exponential backoff.
- Respect the `Retry-After` response header.
- Stop retrying after the 24-hour delivery deadline.
- Move expired events to a dead-letter state.
- Sign webhook requests using HMAC-SHA256.
- Inspect event status and delivery attempt history.
- Recover events left processing after an application interruption.

## Technology Stack

- **Language:** Go
- **Database:** PostgreSQL
- **HTTP:** Go standard library (`net/http`)
- **Database driver:** pgx
- **Event IDs:** UUID
- **Signature:** HMAC-SHA256
- **Containerization:** Docker Compose

## Project Structure

```text
webhook-delivery-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── database/
│   ├── delivery/
│   │   ├── sender.go
│   │   ├── sender_test.go
│   │   ├── signature.go
│   │   └── signature_test.go
│   ├── handlers/
│   │   ├── events.go
│   │   └── events_test.go
│   ├── models/
│   ├── repository/
│   └── worker/
├── migrations/
│   └── 001_create_tables.sql
├── .env.example
├── .gitignore
├── DECISIONS.md
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

## Prerequisites

Install the following:

- Go (version compatible with the project's `go.mod`)
- Docker Desktop
- Git

## Setup and Installation

### 1. Clone the repository

Open PowerShell and run:

```powershell
git clone https://github.com/ChamiNarasinghe/webhook-delivery-service.git
cd webhook-delivery-service
```

### 2. Start PostgreSQL

Make sure Docker Desktop is running, then execute:

```powershell
docker compose up -d
```

Check the database container:

```powershell
docker compose ps
```

The PostgreSQL container should show a healthy status.

### 3. Configure environment variables

The service requires a secret for each configured tenant.

Run these commands in the PowerShell terminal where you will start the Go service:

```powershell
$env:WEBHOOK_SECRET_ACME = "your-acme-secret"
$env:WEBHOOK_SECRET_GLOBEX = "your-globex-secret"
$env:WEBHOOK_SECRET_INITECH = "your-initech-secret"
```

The database connection variable can also be configured:

```powershell
$env:DATABASE_URL = "postgres://webhook_user:webhook_pass@localhost:5432/webhook_db?sslmode=disable"
```

If `DATABASE_URL` is not set, the application uses its configured default connection string.

The `.env.example` file documents the environment variables. The application reads environment variables directly; it does not automatically load `.env.example`.

### 4. Configure tenants

Tenant webhook URLs are configured in:

```text
internal/config/tenants.json
```

Example:

```json
{
  "tenants": {
    "acme": {
      "webhook_url": "http://localhost:9000/hooks/acme"
    },
    "globex": {
      "webhook_url": "http://localhost:9000/hooks/globex"
    },
    "initech": {
      "webhook_url": "http://localhost:9000/hooks/initech"
    }
  }
}
```

Tenant secrets are supplied through environment variables, not stored in this JSON file.

### 5. Download dependencies

From the project root, run:

```powershell
go mod download
```

### 6. Start the service

Run the application:

```powershell
go run ./cmd/server
```

The server starts on:

```text
http://localhost:8080
```

Database migrations are executed automatically when the application starts.

## Running the Mock Receiver

The supplied mock receiver is used to test webhook deliveries.

The mock receiver is provided separately as part of the assignment. Place its folder alongside the `webhook-delivery-service` folder before running the commands below.

The expected folder structure is:

```text
webhook-task/
├── webhook-delivery-service/
└── mock-receiver/
```

Start the mock receiver in a separate PowerShell terminal.

Navigate to its directory:

```powershell
cd ..\mock-receiver
```

Start it in normal mode:

```powershell
go run . -fail-rate 0 -slow-rate 0 -limit-rate 0 -secrets "acme=acme-secret,globex=globex-secret,initech=initech-secret"
```

The mock receiver listens on port `9000`.

Keep this terminal running while testing the delivery service.

**Important:** The mock receiver secrets must match the corresponding webhook secrets configured for the Go service.

## API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| POST | `/events` | Create and queue an event |
| GET | `/events/{id}` | Retrieve event status and attempt history |
| GET | `/dead-letters` | List dead-letter events |

### 1. Create an event

**Request:**

```http
POST http://localhost:8080/events
Content-Type: application/json
```

Example PowerShell request:

```powershell
$body = @{
    tenant_id = "acme"
    type = "order.created"
    payload = @{
        order_id = "ORD-1001"
        amount = 2500
    }
} | ConvertTo-Json -Depth 10

Invoke-RestMethod `
    -Uri "http://localhost:8080/events" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body
```

The service responds with HTTP `202 Accepted` when the event has been successfully queued.

The response includes the generated event ID and its initial status.

### 2. Retrieve an event

Replace `YOUR_EVENT_ID` with the ID returned when creating an event:

```powershell
Invoke-RestMethod `
    -Uri "http://localhost:8080/events/YOUR_EVENT_ID" `
    -Method Get
```

The response includes the event details and its delivery attempt history.

### 3. Retrieve dead-letter events

```powershell
Invoke-RestMethod `
    -Uri "http://localhost:8080/dead-letters" `
    -Method Get
```

The response includes the number of dead-letter events and their details.

## Delivery Behaviour

### Retry Strategy

Failed deliveries are retried using exponential backoff. The delay increases between attempts and is capped at one hour.

When a receiver supplies a valid `Retry-After` value, the service uses that delay.

Events are no longer retried once their 24-hour delivery deadline has been reached. They are marked as `dead` and can be inspected through the dead-letter endpoint.

### Tenant Isolation and Ordering

Each tenant has an independent sequential delivery worker.

Events for the same tenant are processed in creation order, while different tenants can be processed concurrently.

This prevents a slow or unavailable tenant endpoint from blocking deliveries to other tenants.

### Request Signatures

Every outgoing webhook request includes:

- `X-Event-ID`: The unique event identifier.
- `X-Webhook-Signature`: The HMAC-SHA256 signature in `sha256=<hex>` format.
- `Content-Type`: `application/json`.

The signature is generated using the tenant's secret and the exact JSON payload bytes sent in the request.

### Delivery Guarantees

The service provides at-least-once delivery.

A receiver may receive the same event more than once if a delivery succeeds but the service does not record the successful response.

Receivers should use `X-Event-ID` to handle duplicate requests safely through idempotency.

## Running Tests

Run all Go tests from the project root:

```powershell
go test ./...
```

To run tests with detailed output:

```powershell
go test ./... -v
```

## Stopping the Service

Stop the Go application using `Ctrl+C`.

Stop PostgreSQL when it is no longer needed:

```powershell
docker compose stop
```

To start PostgreSQL again:

```powershell
docker compose up -d
```

The database uses a persistent Docker volume, so stopping or replacing the container does not remove the stored database data.

## Security Notes

- Do not commit real webhook secrets or `.env` files.
- Use environment variables or a secrets manager for tenant secrets.
- The credentials in `docker-compose.yml` are for local development and should be replaced with secure credentials in production.
- Use HTTPS for webhook endpoints in production.
