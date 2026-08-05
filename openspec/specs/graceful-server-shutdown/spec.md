# graceful-server-shutdown Specification

## Purpose

Ensures the HTTP server can stop successfully when browser clients hold long-lived event streams open.

## Requirements

### Requirement: Active event streams terminate for server shutdown
The system SHALL terminate active `GET /api/events` streams when server shutdown begins so that server shutdown can complete successfully without waiting for the stream timeout.

#### Scenario: UI client is subscribed during interruption
- **WHEN** a client holds an active `GET /api/events` stream and the server receives a shutdown request
- **THEN** the stream ends, its subscription is released, and the server completes shutdown successfully

### Requirement: Finite requests retain graceful-drain behavior
The system SHALL allow finite in-flight HTTP requests to complete during server shutdown when they finish before the configured shutdown deadline.

#### Scenario: Finite request completes before shutdown deadline
- **WHEN** a finite HTTP request is active when server shutdown begins and the request completes before the shutdown deadline
- **THEN** the request completes normally before the server exits
