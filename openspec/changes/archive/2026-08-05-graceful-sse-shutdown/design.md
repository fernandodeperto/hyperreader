## Context

`http.Server.Shutdown` closes listeners and idle connections but waits for active handlers. The embedded UI creates an EventSource connection to `GET /api/events`, and that handler normally runs until the client disconnects. See proposal.md for motivation and `graceful-server-shutdown` for required behavior.

## Goals / Non-Goals

**Goals:**
- End active SSE handlers as shutdown starts.
- Preserve graceful draining for ordinary finite requests.
- Keep event subscriptions from leaking during shutdown.
- Cover the live SSE shutdown path with a real network-level test.

**Non-Goals:**
- Change the `GET /api/events` protocol or browser EventSource behavior.
- Add a write timeout to HTTP responses.
- Force-close ordinary in-flight requests before the shutdown deadline.
- Alter startup, storage, or signal registration behavior.

## Decisions

### Use an explicit SSE lifecycle signal

The server will own a one-way shutdown signal and trigger it immediately before invoking `http.Server.Shutdown`. Event-stream handlers will select on this signal alongside client request cancellation and return when it is triggered. Their existing deferred cleanup will release each subscription.

This scopes early cancellation to a deliberately unbounded request type. Cancelling the server's parent request context instead was rejected because it could abort finite in-flight requests rather than draining them. Relying on `http.Server.Shutdown` alone was rejected because active SSE handlers never become idle by themselves.

The signal remains closed after shutdown begins, so an event handler admitted in the narrow interval before the listener closes also exits promptly.

### Keep the HTTP shutdown deadline as the final safety bound

After event streams are signalled, the server will still use `http.Server.Shutdown` with its existing deadline. This retains bounded shutdown for stuck finite requests while allowing ordinary requests that finish in time to complete normally.

Ignoring a shutdown deadline error or switching immediately to `http.Server.Close` was rejected because either approach hides or causes an ungraceful termination.

### Verify through the real server lifecycle

A server integration test will open an actual SSE connection, begin shutdown through `server.Run`'s context, and assert a prompt successful return. Existing no-client shutdown coverage remains the control case. The test will not depend on the browser UI because the HTTP event-stream contract is sufficient to reproduce the lifecycle condition.

## Risks / Trade-offs

- [An SSE client observes a disconnected stream] → This is the required shutdown behavior. Browser EventSource reconnection attempts end when the process exits.
- [A new SSE connection races with shutdown] → Keep the lifecycle signal permanently closed once shutdown begins so that a newly admitted handler exits immediately.
- [A future long-lived endpoint is added] → It must opt into equivalent lifecycle handling rather than relying on the finite-request shutdown path.
