## 1. Fluid application shell

- [x] 1.1 Remove the fixed maximum width from the application shell while retaining its existing responsive page gutters.
- [x] 1.2 Confirm the header, toolbar, empty and error states, and documents table continue to fill the shell without changing content-sized controls.

## 2. Browser verification

- [x] 2.1 Add or update a Playwright viewport assertion that proves the documents table uses substantially more than the former fixed-width layout at a wide viewport.
- [x] 2.2 Exercise the UI at a narrow viewport and verify the shell preserves gutters without horizontal viewport overflow.
