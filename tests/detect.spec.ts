import { test, expect } from "@playwright/test";
import { gotoApp, makeGate, STATE_ROUTE, DETECT_ROUTE } from "./helpers";

// The "偵測裝置" button posts /api/detect, which runs two nrfutil commands and
// reports two independent checks: the J-Link probe (device list) and the target
// MCU behind it (device device-info). Mirrors the detectButton click handler
// and renderDetectCheck() in cmd/nrf-factory/web/index.html.

function detectResponse(jlink: { found: boolean; message: string }, mcu: { found: boolean; message: string }, serials: string[] = []) {
  return {
    status: 200,
    contentType: "application/json; charset=utf-8",
    body: JSON.stringify({ jlink, mcu, serials }),
  };
}

test.describe("detect device", () => {
  test("J-Link and MCU both present show two found rows", async ({ page }) => {
    await gotoApp(page);

    await page.route(DETECT_ROUTE, (route) =>
      route.fulfill(
        detectResponse(
          { found: true, message: "J-Link 已連接：000802009570" },
          { found: true, message: "MCU 已連接" },
          ["000802009570"],
        ),
      ),
    );

    await page.locator("#detectButton").click();

    await expect(page.locator("#detectJLink")).toHaveText("J-Link 已連接：000802009570");
    await expect(page.locator("#detectJLink")).toHaveClass(/found/);
    await expect(page.locator("#detectMCU")).toHaveText("MCU 已連接");
    await expect(page.locator("#detectMCU")).toHaveClass(/found/);
    await expect(page.locator(".log-line", { hasText: "MCU 已連接" })).toHaveClass(/success/);
  });

  test("J-Link present but MCU absent shows found and missing separately", async ({ page }) => {
    await gotoApp(page);

    await page.route(DETECT_ROUTE, (route) =>
      route.fulfill(
        detectResponse(
          { found: true, message: "J-Link 已連接：000802009570" },
          { found: false, message: "讀不到 MCU（晶片未連接或未供電）" },
          ["000802009570"],
        ),
      ),
    );

    await page.locator("#detectButton").click();

    await expect(page.locator("#detectJLink")).toHaveClass(/found/);
    await expect(page.locator("#detectMCU")).toContainText("讀不到 MCU");
    await expect(page.locator("#detectMCU")).toHaveClass(/missing/);
    await expect(page.locator(".log-line", { hasText: "讀不到 MCU" })).toHaveClass(/error/);
  });

  test("no J-Link marks both rows missing", async ({ page }) => {
    await gotoApp(page);

    await page.route(DETECT_ROUTE, (route) =>
      route.fulfill(
        detectResponse(
          { found: false, message: "未偵測到 J-Link，請確認 USB 連接" },
          { found: false, message: "無法偵測 MCU（需先接上 J-Link）" },
        ),
      ),
    );

    await page.locator("#detectButton").click();

    await expect(page.locator("#detectJLink")).toContainText("未偵測到 J-Link");
    await expect(page.locator("#detectJLink")).toHaveClass(/missing/);
    await expect(page.locator("#detectMCU")).toContainText("需先接上 J-Link");
    await expect(page.locator("#detectMCU")).toHaveClass(/missing/);
  });

  test("multiple J-Links warn on the J-Link row without an MCU result", async ({ page }) => {
    await gotoApp(page);

    await page.route(DETECT_ROUTE, (route) =>
      route.fulfill(
        detectResponse(
          { found: false, message: "偵測到多個 J-Link，燒錄前請只保留一個：AAA, BBB" },
          { found: false, message: "無法偵測 MCU（請只保留一個 J-Link）" },
          ["AAA", "BBB"],
        ),
      ),
    );

    await page.locator("#detectButton").click();

    await expect(page.locator("#detectJLink")).toContainText("多個 J-Link");
    await expect(page.locator("#detectJLink")).toHaveClass(/missing/);
    await expect(page.locator("#detectMCU")).toContainText("只保留一個");
    await expect(page.locator("#detectMCU")).toHaveClass(/missing/);
  });

  test("busy lock: detecting disables inputs mid-request", async ({ page }) => {
    await gotoApp(page);

    const gate = makeGate();
    await page.route(DETECT_ROUTE, async (route) => {
      await gate.opened;
      await route.fulfill(
        detectResponse(
          { found: true, message: "J-Link 已連接：000802009570" },
          { found: true, message: "MCU 已連接" },
          ["000802009570"],
        ),
      );
    });

    await page.locator("#detectButton").click();

    await expect(page.locator("#detectButton")).toBeDisabled();
    await expect(page.locator("#detectJLink")).toHaveText("J-Link：偵測中…");
    await expect(page.locator("#detectMCU")).toHaveText("MCU：偵測中…");
    await expect(page.locator("body")).toHaveClass(/busy/);
    await expect(page.locator("#firmware")).toBeDisabled();

    gate.open();

    await expect(page.locator("#detectJLink")).toHaveClass(/found/);
    await expect(page.locator("#detectMCU")).toHaveClass(/found/);
    await expect(page.locator("body")).not.toHaveClass(/busy/);
    await expect(page.locator("#detectButton")).toBeEnabled();
  });

  test("detect HTTP error marks both rows as failed", async ({ page }) => {
    await gotoApp(page);

    await page.route(DETECT_ROUTE, (route) =>
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: "boom" }),
      }),
    );

    await page.locator("#detectButton").click();

    await expect(page.locator("#detectJLink")).toHaveText("J-Link：偵測失敗");
    await expect(page.locator("#detectJLink")).toHaveClass(/missing/);
    await expect(page.locator("#detectMCU")).toHaveText("MCU：偵測失敗");
    await expect(page.locator("#detectMCU")).toHaveClass(/missing/);
    await expect(page.locator(".log-line.error")).toContainText("boom");
  });

  test("detect button is disabled when the server is offline", async ({ page }) => {
    await page.route(STATE_ROUTE, (route) => route.abort("failed"));
    await page.goto("/");

    await expect(page.locator("#toolMessage")).toContainText("無法連線本機服務");
    await expect(page.locator("#detectButton")).toBeDisabled();
  });
});
