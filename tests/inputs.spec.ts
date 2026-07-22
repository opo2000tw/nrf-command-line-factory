import { test, expect } from "@playwright/test";
import { gotoApp, firmwareFile } from "./helpers";

// Firmware file input + L/R station selection + flash button enable/label.
// Mirrors the firmwareInput "change" handler, the station click handler, and
// refreshFlashButton() in cmd/nrf-factory/web/index.html.
test.describe("firmware input + side selection", () => {
  test("initial state: flash disabled, L selected", async ({ page }) => {
    await gotoApp(page);

    await expect(page.locator("#flashButton")).toBeDisabled();
    await expect(page.locator("#flashButton")).toHaveText("開始燒錄 L");
    await expect(page.locator('.station[data-side="L"]')).toHaveClass(/active/);
    await expect(page.locator('.station[data-side="L"]')).toHaveAttribute("aria-pressed", "true");
    await expect(page.locator('.station[data-side="R"]')).toHaveAttribute("aria-pressed", "false");
  });

  test("selecting a .hex file shows the name, hint, and enables flash", async ({ page }) => {
    await gotoApp(page);

    await page.setInputFiles("#firmware", firmwareFile("blink.hex"));

    await expect(page.locator("#fileName")).toHaveText("blink.hex");
    await expect(page.locator("#fileHint")).toContainText("KB · 僅本次作業");
    await expect(page.locator("#flashButton")).toBeEnabled();
  });

  test("fileHint rounds size up to whole KB", async ({ page }) => {
    await gotoApp(page);

    // Default fixture body is ~12 bytes -> ceil(12/1024) = 1, floored at 1 by Math.max.
    await page.setInputFiles("#firmware", firmwareFile());
    await expect(page.locator("#fileHint")).toHaveText(/^1 KB/);

    // 3000 bytes -> ceil(3000/1024) = 3.
    await page.setInputFiles("#firmware", firmwareFile("big.hex", "x".repeat(3000)));
    await expect(page.locator("#fileHint")).toHaveText(/^3 KB/);
  });

  test("clicking R activates R without touching the counters", async ({ page }) => {
    await gotoApp(page);

    await page.locator('.station[data-side="R"]').click();

    await expect(page.locator('.station[data-side="R"]')).toHaveClass(/active/);
    await expect(page.locator('.station[data-side="R"]')).toHaveAttribute("aria-pressed", "true");
    await expect(page.locator('.station[data-side="L"]')).toHaveAttribute("aria-pressed", "false");
    await expect(page.locator("#flashButton")).toHaveText("開始燒錄 R");
    await expect(page.locator("#leftCount")).toHaveText("0");
    await expect(page.locator("#rightCount")).toHaveText("0");
  });

  test("clicking R then back to L restores L selection", async ({ page }) => {
    await gotoApp(page);

    await page.locator('.station[data-side="R"]').click();
    await page.locator('.station[data-side="L"]').click();

    await expect(page.locator('.station[data-side="L"]')).toHaveClass(/active/);
    await expect(page.locator('.station[data-side="L"]')).toHaveAttribute("aria-pressed", "true");
    await expect(page.locator('.station[data-side="R"]')).toHaveAttribute("aria-pressed", "false");
    await expect(page.locator("#flashButton")).toHaveText("開始燒錄 L");
  });

  test("selecting a side before choosing firmware still enables flash correctly", async ({ page }) => {
    await gotoApp(page);

    await page.locator('.station[data-side="R"]').click();
    await page.setInputFiles("#firmware", firmwareFile());

    await expect(page.locator("#flashButton")).toBeEnabled();
    await expect(page.locator("#flashButton")).toHaveText("開始燒錄 R");
  });
});
