import { test, expect } from "@playwright/test";
import { gotoApp, defaultState, ndjson, toolFile, makeGate, TOOL_ROUTE, INSTALL_ROUTE } from "./helpers";

// Tool-setup panel: uploading an nrfutil executable (#toolFile -> POST /api/tool),
// installing the device command (#installButton -> POST /api/install-device), and
// the resulting tool status / command-path display. Mirrors the toolFile "change"
// handler, the installButton click handler, and the command-path branch of
// renderState() in cmd/nrf-factory/web/index.html.
test.describe("tool setup", () => {
  test("uploading a tool file succeeds and shows the ready state", async ({ page }) => {
    await gotoApp(page, {
      toolReady: false,
      toolMessage: "找不到 nrfutil，請先安裝 Nordic nRF Util 與 device command",
    });

    await page.route(TOOL_ROUTE, (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          defaultState({
            toolReady: true,
            toolMessage: "nrfutil 7.13.0 · device 2.7.6 · SEGGER J-Link V9.60 OK",
            command: "/tmp/nrf-factory-nrfutil",
          }),
        ),
      }),
    );

    await page.setInputFiles("#toolFile", toolFile("nrfutil"));

    await expect(page.locator(".log-line", { hasText: "指定 nrfutil：nrfutil，驗證中…" })).toHaveClass(/info/);
    await expect(page.locator(".log-line.success")).toContainText("工具就緒：");
    await expect(page.locator("#toolStatus")).toHaveClass(/ready/);
    await expect(page.locator("#toolMessage")).toContainText("SEGGER J-Link V9.60 OK");
    // renderState overwrites the picked filename with the resolved command's basename.
    await expect(page.locator("#toolFileName")).toHaveText("nrf-factory-nrfutil");
    await expect(page.locator("#toolFileHint")).toHaveText("/tmp/nrf-factory-nrfutil");
  });

  test("uploading a tool file that is still missing logs the shortfall", async ({ page }) => {
    await gotoApp(page, {
      toolReady: false,
      toolMessage: "找不到 nrfutil，請先安裝 Nordic nRF Util 與 device command",
    });

    await page.route(TOOL_ROUTE, (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          defaultState({
            toolReady: false,
            toolMessage: "缺少 nrfutil device command；請執行 nrfutil install device",
            command: "nrfutil",
          }),
        ),
      }),
    );

    await page.setInputFiles("#toolFile", toolFile());

    await expect(page.locator(".log-line.error")).toContainText("仍缺：");
    await expect(page.locator("#toolStatus")).not.toHaveClass(/ready/);
  });

  test("HTTP error response while uploading logs the server's error message", async ({ page }) => {
    await gotoApp(page);

    await page.route(TOOL_ROUTE, (route) =>
      route.fulfill({
        status: 400,
        contentType: "application/json",
        body: JSON.stringify({ error: "無法儲存 nrfutil 執行檔" }),
      }),
    );

    await page.setInputFiles("#toolFile", toolFile());

    await expect(page.locator(".log-line.error")).toContainText("無法儲存 nrfutil 執行檔");
  });

  test("installing the device command succeeds and shows the ready state", async ({ page }) => {
    await gotoApp(page, {
      toolReady: false,
      deviceReady: false,
      toolMessage: "缺少 nrfutil device command；請執行 nrfutil install device",
    });

    await page.route(INSTALL_ROUTE, (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/x-ndjson; charset=utf-8",
        body: ndjson([
          { level: "info", message: "安裝 nrfutil device 命令…" },
          {
            level: "success",
            message: "device 命令安裝完成",
            done: true,
            success: true,
            state: defaultState({ toolReady: true, toolMessage: "nrfutil 7.13.0 · device 2.7.6 · SEGGER J-Link V9.60 OK" }),
          },
        ]),
      }),
    );

    await page.locator("#installButton").click();

    await expect(page.locator("#terminal")).toContainText("開始安裝 nrfutil device 命令…（下載需一點時間）");
    await expect(page.locator(".log-line.success")).toContainText("device 命令安裝完成");
    await expect(page.locator("#toolStatus")).toHaveClass(/ready/);
    await expect(page.locator("#toolMessage")).toContainText("SEGGER J-Link V9.60 OK");
    // Now that the device command is installed, the button greys out.
    await expect(page.locator("#installButton")).toBeDisabled();
    await expect(page.locator("#installButton")).toHaveText("device 命令已安裝");
  });

  test("busy lock: installing disables inputs mid-request, then releases on completion", async ({ page }) => {
    await gotoApp(page, { deviceReady: false });

    // No pre-ping before POSTing /api/install-device (unlike flash/tool), so the
    // busy lock is driven purely by the gated response below.
    const gate = makeGate();
    await page.route(INSTALL_ROUTE, async (route) => {
      await gate.opened;
      await route.fulfill({
        status: 200,
        contentType: "application/x-ndjson; charset=utf-8",
        body: ndjson([
          {
            level: "success",
            message: "device 命令安裝完成",
            done: true,
            success: true,
            state: defaultState(),
          },
        ]),
      });
    });

    await page.locator("#installButton").click();

    await expect(page.locator("#installButton")).toBeDisabled();
    await expect(page.locator("#firmware")).toBeDisabled();
    await expect(page.locator("#resetButton")).toBeDisabled();
    await expect(page.locator('.station[data-side="L"]')).toBeDisabled();
    await expect(page.locator('.station[data-side="R"]')).toBeDisabled();
    await expect(page.locator("body")).toHaveClass(/busy/);

    gate.open();

    await expect(page.locator("body")).not.toHaveClass(/busy/);
    await expect(page.locator("#firmware")).toBeEnabled();
  });

  test("installing fails with an HTTP error and logs the message", async ({ page }) => {
    await gotoApp(page, { deviceReady: false });

    await page.route(INSTALL_ROUTE, (route) =>
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: "安裝失敗：network down" }),
      }),
    );

    await page.locator("#installButton").click();

    await expect(page.locator(".log-line.error")).toContainText("安裝失敗：network down");
  });

  test("install button is greyed out when the device command is already installed", async ({ page }) => {
    await gotoApp(page, { deviceReady: true });

    await expect(page.locator("#installButton")).toBeDisabled();
    await expect(page.locator("#installButton")).toHaveText("device 命令已安裝");
  });

  test("install button is enabled when the device command is missing", async ({ page }) => {
    await gotoApp(page, {
      toolReady: false,
      deviceReady: false,
      toolMessage: "缺少 nrfutil device command；請執行 nrfutil install device",
    });

    await expect(page.locator("#installButton")).toBeEnabled();
    await expect(page.locator("#installButton")).toHaveText("安裝 device 命令");
  });

  test("loading with a bundled 3rd/ command shows the bundled hint", async ({ page }) => {
    await gotoApp(page, { command: "/opt/app/3rd/nrfutil" });

    await expect(page.locator("#toolFileName")).toHaveText("nrfutil");
    await expect(page.locator("#toolFileHint")).toContainText("使用內附 nrfutil（3rd/）");
  });
});
