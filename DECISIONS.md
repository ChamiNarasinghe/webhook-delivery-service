# Design Decisions

## 1. PostgreSQL as a Persistent Queue

PostgreSQL stores events and delivery attempts instead of relying on in-memory queues. This ensures that events and their delivery history survive application restarts. A database-backed queue was chosen to keep the implementation simple while providing durable storage.

## 2. At-Least-Once Delivery

The service uses at-least-once delivery rather than claiming exactly-once delivery. If a receiver processes an event but the service fails before recording the successful response, the event may be delivered again. A stable event ID is included in every request so receivers can implement idempotency.

## 3. Per-Tenant Sequential Workers

Each tenant has an independent worker that processes one event at a time. This preserves event creation order within a tenant while allowing different tenants to receive events concurrently. PostgreSQL advisory transaction locks protect tenant-level event claiming.

## 4. Retry and Expiry Strategy

Failed deliveries use exponential backoff, starting at one second and capped at one hour. The service also respects the receiver's `Retry-After` header. Each event has a 24-hour delivery deadline, after which it is marked as dead and made available through the dead-letter endpoint.

## 5. Processing Recovery

Events are marked as processing before delivery. A processing lease allows the service to recover events that remain in processing after an unexpected interruption. Events with expired deadlines are moved to the dead-letter state.

## 6. Webhook Security

Outgoing requests are signed using HMAC-SHA256 and the tenant-specific secret. The signature is calculated from the exact payload bytes sent to the receiver. Secrets are loaded from environment variables rather than stored in the tenant configuration file.

## 7. HTTP Timeout

A 10-second HTTP client timeout prevents a slow or unresponsive receiver from occupying a tenant worker indefinitely. This helps maintain isolation between tenants.

## 8. Ambiguities and Trade-offs

- Failed responses, including HTTP 410, are treated as retryable until the 24-hour deadline because the assignment specifies retrying failed deliveries but does not define special handling for individual status codes.
- The processing recovery lease is 30 seconds. This is longer than the configured HTTP timeout, reducing the risk of another worker recovering an event while its original delivery is still running.
- The service prioritizes a straightforward PostgreSQL-based implementation over introducing an external message broker.

## 9. Production Improvements

For production use, I would consider:

- Adding authentication, authorization, and rate limiting to the API.
- Storing secrets in a dedicated secrets manager.
- Adding retry jitter to reduce simultaneous retry bursts.
- Improving worker coordination and lease management for multi-instance deployments.
- Adding monitoring, metrics, distributed tracing, and structured logging.
- Providing administrative tools for dead-letter inspection and replay.
- Enforcing HTTPS for webhook endpoints and securely managing database credentials.