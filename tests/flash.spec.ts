import { test, expect } from "@playwright/test";
import { gotoApp, defaultState, ndjson, firmwareFile, makeGate, FLASH_ROUTE } from "./helpers";

// POST /api/flash flow: success (L/R), HTTP error, streamed error event, the
// busy lock while a flash is in flight, and the offline/network-failure path.
// Mirrors the flashButton click handler, readEvents/handleEvent, and
// networkHint() in cmd/nrf-factory/web/index.html.
test.describe("flash", () => {
  test("successful flash on L logs the trace and bumps the left counter", async ({ page }) => {
    await gotoApp(page);
    await page.setInputFiles("#firmware", firmwareFile());

    await page.route(FLASH_ROUTE, (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/x-ndjson; charset=utf-8",
        body: ndjson([
          { level: "info", message: "J-Link 000680123456" },
          {
            level: "success",
            message: "L 側燒錄成功",
            done: true,
            success: true,
            state: defaultState({ leftCount: 1 }),
          },
        ]),
      }),
    );

    await page.locator("#flashButton").click();

    await expect(page.locator("#leftCount")).toHaveText("1");
    await expect(page.locator(".log-line", { hasText: "L 側燒錄成功" })).toHaveClass(/success/);
    await expect(page.locator("#terminal")).toContainText("J-Link 000680123456");
  });

  test("successful flash on R logs the trace and bumps the right counter", async ({ page }) => {
    await gotoApp(page);
    await page.locator('.station[data-side="R"]').click();
    await page.setInputFiles("#firmware", firmwareFile());

    await page.route(FLASH_ROUTE, (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/x-ndjson; charset=utf-8",
        body: ndjson([
          {
            level: "success",
            message: "R 側燒錄成功",
            done: true,
            success: true,
            state: defaultState({ rightCount: 1 }),
          },
        ]),
      }),
    );

    await page.locator("#flashButton").click();

    await expect(page.locator("#rightCount")).toHaveText("1");
    await expect(page.locator(".log-line", { hasText: "R 側燒錄成功" })).toHaveClass(/success/);
  });

  test("HTTP error response logs the server's error message and leaves counters untouched", async ({ page }) => {
    await gotoApp(page);
    await page.setInputFiles("#firmware", firmwareFile());

    await page.route(FLASH_ROUTE, (route) =>
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: "nrfutil program: 找不到 J-Link" }),
      }),
    );

    await page.locator("#flashButton").click();

    await expect(page.locator(".log-line.error")).toContainText("nrfutil program: 找不到 J-Link");
    await expect(page.locator("#leftCount")).toHaveText("0");
  });

  test("streamed error event (200 + done/success:false) logs failure and leaves counters untouched", async ({ page }) => {
    await gotoApp(page);
    await page.setInputFiles("#firmware", firmwareFile());

    await page.route(FLASH_ROUTE, (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/x-ndjson; charset=utf-8",
        body: ndjson([
          {
            level: "error",
            message: "燒錄失敗：找不到 J-Link",
            done: true,
            success: false,
            state: defaultState(),
          },
        ]),
      }),
    );

    await page.locator("#flashButton").click();

    await expect(page.locator(".log-line.error")).toContainText("燒錄失敗");
    await expect(page.locator("#leftCount")).toHaveText("0");
  });

  test("busy lock: flash disables inputs mid-request, then releases on completion", async ({ page }) => {
    const state = await gotoApp(page);
    await page.setInputFiles("#firmware", firmwareFile());

    // The click handler pings GET /api/state BEFORE POSTing /api/flash, and
    // renderState() unconditionally applies that ping's busy flag. Unless the
    // live state already reports busy here, the ping would flip the UI back
    // to unlocked while the (still in-flight) POST is gated below.
    state.busy = true;

    const gate = makeGate();
    await page.route(FLASH_ROUTE, async (route) => {
      await gate.opened;
      await route.fulfill({
        status: 200,
        contentType: "application/x-ndjson; charset=utf-8",
        body: ndjson([
          {
            level: "success",
            message: "L 側燒錄成功",
            done: true,
            success: true,
            state: { ...state },
          },
        ]),
      });
    });

    await page.locator("#flashButton").click();

    await expect(page.locator("#flashButton")).toBeDisabled();
    await expect(page.locator("#flashButton")).toHaveText("正在燒錄 L");
    await expect(page.locator("body")).toHaveClass(/busy/);
    await expect(page.locator("#firmware")).toBeDisabled();
    await expect(page.locator('.station[data-side="L"]')).toBeDisabled();
    await expect(page.locator('.station[data-side="R"]')).toBeDisabled();

    state.busy = false;
    state.leftCount = 1;
    gate.open();

    await expect(page.locator("#leftCount")).toHaveText("1");
    await expect(page.locator("#flashButton")).toHaveText("開始燒錄 L");
    await expect(page.locator("body")).not.toHaveClass(/busy/);
  });

  test("aborted /api/flash request surfaces the offline network hint", async ({ page }) => {
    await gotoApp(page);
    await page.setInputFiles("#firmware", firmwareFile());

    await page.route(FLASH_ROUTE, (route) => route.abort("failed"));

    await page.locator("#flashButton").click();

    await expect(page.locator(".log-line.error")).toContainText("Failed to fetch");
    await expect(page.locator("#leftCount")).toHaveText("0");
  });
});
