// Playwright config for the html-mcp browser smoke.
//
// The smoke runs against the REAL `html-mcp serve` binary (not a Node mock
// of the API): webServer builds the Go binary and starts it on a free port
// with a throwaway data dir, so the test exercises the same composed mux
// (API at /api/ + embedded UI at /) that production serves. Playwright
// manages the process lifecycle — it starts it before the suite and tears
// it down after.
//
// Browsers: chromium only. This is the first UI slice (S02); broadening to
// firefox/webkit is deferred until the UI stabilizes. The Playwright
// browser binaries are expected to be pre-installed under the default
// ~/Library/Caches/ms-playwright cache (no `install` step needed here).
import { defineConfig, devices } from "@playwright/test";

// PORT is injected by the smoke harness so the test and the webServer share
// one port without hardcoding. Default to 7421 (one off production's 7420)
// for direct `npx playwright test` runs.
const port = Number(process.env.PORT) || 7421;
const baseURL = `http://127.0.0.1:${port}`;

// DATA_DIR is a throwaway data directory under the repo (gitignored). The
// webServer command wipes it before each start so every `npx playwright
// test` run begins on a clean slate — test 1 asserts exact row counts, so a
// leftover store from a prior run must never be reused.
const DATA_DIR = "./.e2e-data";

export default defineConfig({
  testDir: "./e2e",
  // Fail fast on the first browser error — this is a smoke suite, not a
  // full regression matrix.
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: "list",
  use: {
    baseURL,
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  // webServer builds and launches the real serve binary. Playwright polls
  // baseURL until it responds (timeout below), then runs the tests, then
  // kills the process. The data dir is wiped before each start so every run
  // begins on a clean slate (test 1 asserts exact row counts). We never
  // reuse an existing server: a warm reused server would carry leftover
  // docs from the prior run and break the exact-count assertion. Fresh
  // start every run keeps the smoke reliable.
  webServer: {
    command: `rm -rf ${DATA_DIR} && go run . serve --port ${port} --data-dir ${DATA_DIR}`,
    url: baseURL,
    reuseExistingServer: false,
    timeout: 60_000,
    cwd: ".",
  },
});
