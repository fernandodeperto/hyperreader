## 1. SSE shutdown lifecycle

- [x] 1.1 Add a one-way shutdown notification to the event-stream routing path so active and newly admitted `GET /api/events` handlers exit and unregister when shutdown begins.
- [x] 1.2 Trigger the event-stream shutdown notification immediately before the existing bounded `http.Server.Shutdown` call, without cancelling ordinary request contexts.

## 2. Lifecycle verification

- [x] 2.1 Add real server lifecycle tests that prove an SSE subscription is released and a finite active request drains when shutdown begins.
- [x] 2.2 Run the focused server and API test packages to verify SSE behavior, normal shutdown behavior, and the new active-stream shutdown path.
