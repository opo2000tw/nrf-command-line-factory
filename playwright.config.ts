import { defineConfig, devices } from "@playwright/test";

// The Go app serves the embedded web/index.html. We run it with -no-browser on a
// fixed loopback port and let each test intercept /api/* so no nrfutil / J-Link
// hardware is required. `exec` replaces the build shell with the binary so
// Playwright's teardown kills the server process directly.
const HOST = "127.0.0.1";
const PORT = 17832;
const baseURL = `http://${HOST}:${PORT}`;

export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL,
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: `go build -o tests/.bin/nrf-factory ./cmd/nrf-factory && exec tests/.bin/nrf-factory -no-browser -addr ${HOST}:${PORT}`,
    url: baseURL,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    stdout: "pipe",
    stderr: "pipe",
  },
});
