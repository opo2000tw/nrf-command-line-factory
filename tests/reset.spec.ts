import { test, expect } from "@playwright/test";
import { gotoApp, defaultState, STATE_ROUTE, RESET_ROUTE } from "./helpers";

// POST /api/reset flow: success, HTTP conflict (409, flash in progress), and
// the resetButton's disabled state while busy or offline. Mirrors the
// resetButton click handler and setBusy() in cmd/nrf-factory/web/index.html.
test.describe("reset counters", () => {
  test("clears the L/R counters and logs success", async ({ page }) => {
    await gotoApp(page, { leftCount: 4, rightCount: 2 });

    await page.route(RESET_ROUTE, (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(defaultState()),
      }),
    );

    await page.locator("#resetButton").click();

    await expect(page.locator("#leftCount")).toHaveText("0");
    await expect(page.locator("#rightCount")).toHaveText("0");
    await expect(page.locator("#terminal")).toContainText("L/R 成功計數已清除");
  });

  test("HTTP conflict while flashing logs the error and leaves counters untouched", async ({ page }) => {
    await gotoApp(page, { leftCount: 4, rightCount: 2 });

    await page.route(RESET_ROUTE, (route) =>
      route.fulfill({
        status: 409,
        contentType: "application/json",
        body: JSON.stringify({ error: "燒錄進行中，無法清除計數" }),
      }),
    );

    await page.locator("#resetButton").click();

    await expect(
      page.locator(".log-line", { hasText: "燒錄進行中，無法清除計數" }),
    ).toHaveClass(/error/);
    await expect(page.locator("#leftCount")).toHaveText("4");
    await expect(page.locator("#rightCount")).toHaveText("2");
  });

  test("reset button is disabled while busy", async ({ page }) => {
    await gotoApp(page, { busy: true });

    await expect(page.locator("#resetButton")).toBeDisabled();
    await expect(page.locator("body")).toHaveClass(/busy/);
  });

  test("offline on load locks reset (and flash + install) together", async ({ page }) => {
    // markOffline() re-applies the disabled state via setBusy(), so every
    // server-dependent action locks when the backend is unreachable — reset and
    // install now lock alongside flash instead of staying clickable.
    await page.route(STATE_ROUTE, (route) => route.abort("failed"));
    await page.goto("/");

    await expect(page.locator("#toolMessage")).toContainText("無法連線本機服務");
    await expect(page.locator("#flashButton")).toBeDisabled();
    await expect(page.locator("#resetButton")).toBeDisabled();
    await expect(page.locator("#installButton")).toBeDisabled();
  });
});
