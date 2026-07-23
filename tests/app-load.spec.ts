import { test, expect } from "@playwright/test";
import { gotoApp, defaultState, STATE_ROUTE } from "./helpers";

// Initial render + tool-status wiring. Owned by the harness; also acts as the
// smoke test that proves the mock contract (gotoApp / defaultState) works.
test.describe("app load", () => {
  test("renders the ready state from /api/state", async ({ page }) => {
    await gotoApp(page, {
      leftCount: 3,
      rightCount: 5,
      toolReady: true,
      toolMessage: "nrfutil 7.13.0 · device 2.7.6 · SEGGER J-Link V9.60 OK",
      version: "v9.9.9",
    });

    await expect(page.locator("#leftCount")).toHaveText("3");
    await expect(page.locator("#rightCount")).toHaveText("5");
    await expect(page.locator("#toolStatus")).toHaveClass(/ready/);
    await expect(page.locator("#toolMessage")).toContainText("SEGGER J-Link V9.60 OK");
    await expect(page.locator("#appVersion")).toHaveText("v9.9.9");
  });

  test("shows the not-ready tool status without the ready class", async ({ page }) => {
    await gotoApp(page, {
      toolReady: false,
      toolMessage: "找不到 nrfutil，請先安裝 Nordic nRF Util 與 device command",
    });

    await expect(page.locator("#toolStatus")).not.toHaveClass(/ready/);
    await expect(page.locator("#toolMessage")).toContainText("找不到 nrfutil");
  });

  test("goes offline when /api/state is unreachable", async ({ page }) => {
    // Do not use gotoApp here: we want the state route to fail from the start.
    await page.route(STATE_ROUTE, (route) => route.abort("failed"));
    await page.goto("/");

    await expect(page.locator("#toolMessage")).toContainText("無法連線本機服務");
    await expect(page.locator("#terminal")).toContainText("Failed to fetch");
    // Offline => every server-dependent action locks, not just flash.
    await expect(page.locator("#flashButton")).toBeDisabled();
    await expect(page.locator("#resetButton")).toBeDisabled();
    await expect(page.locator("#installButton")).toBeDisabled();
  });

  test("flash button starts disabled with the selected side in its label", async ({ page }) => {
    await gotoApp(page);
    await expect(page.locator("#flashButton")).toBeDisabled();
    await expect(page.locator("#flashButton")).toHaveText("開始燒錄 L");
    // Sanity: defaultState is the shared baseline the other specs build on.
    expect(defaultState().leftCount).toBe(0);
  });
});
