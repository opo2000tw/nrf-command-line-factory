import { Page } from "@playwright/test";

// Shared test contract for the nRF Factory UI. Every spec drives the real
// embedded index.html served by the Go app, but intercepts /api/* so no nrfutil
// or J-Link hardware is touched. See patterns at the bottom of this file.

export const STATE_ROUTE = "**/api/state";
export const FLASH_ROUTE = "**/api/flash";
export const RESET_ROUTE = "**/api/reset";
export const TOOL_ROUTE = "**/api/tool";
export const INSTALL_ROUTE = "**/api/install-device";

// Mirror of the Go `snapshot` struct rendered by renderState() in index.html.
export type AppState = {
  leftCount: number;
  rightCount: number;
  busy: boolean;
  toolReady: boolean;
  toolMessage: string;
  command: string;
  version: string;
};

// Mirror of the Go `apiEvent` struct emitted line-by-line by /api/flash and
// /api/install-device (NDJSON: one JSON object per line).
export type ApiEvent = {
  level: "info" | "success" | "error";
  message: string;
  done?: boolean;
  success?: boolean;
  state?: AppState;
};

export function defaultState(overrides: Partial<AppState> = {}): AppState {
  return {
    leftCount: 0,
    rightCount: 0,
    busy: false,
    toolReady: true,
    toolMessage: "nrfutil 7.13.0 · device 2.7.6 · J-Link OK",
    command: "nrfutil",
    version: "test-1.2.3",
    ...overrides,
  };
}

// Serialize apiEvents into the NDJSON body /api/flash + /api/install-device send.
export function ndjson(events: ApiEvent[]): string {
  return events.map((event) => JSON.stringify(event)).join("\n") + "\n";
}

// In-memory firmware upload; avoids committing a .hex fixture (gitignored anyway).
export function firmwareFile(name = "firmware.hex", body = ":00000001FF\n") {
  return { name, mimeType: "application/octet-stream", buffer: Buffer.from(body) };
}

// In-memory nrfutil upload for the tool picker.
export function toolFile(name = "nrfutil", body = "#!/bin/sh\necho nrfutil\n") {
  return { name, mimeType: "application/octet-stream", buffer: Buffer.from(body) };
}

// A one-shot gate to hold a mocked response open so busy/disabled states are
// observable mid-request. Await `opened` inside a route handler; call `open()`
// from the test once the assertions are done.
export function makeGate(): { opened: Promise<void>; open: () => void } {
  let open!: () => void;
  const opened = new Promise<void>((resolve) => {
    open = resolve;
  });
  return { opened, open };
}

// Navigate to the app with /api/state stubbed by a LIVE, mutable state object.
// The returned object is read on every poll (initial load + the 3s interval +
// the pre-flash ping), so mutating it (e.g. `state.busy = true`) changes what
// the server "reports" from that point on. Resolves once the initial render has
// applied the state (tool message painted).
export async function gotoApp(
  page: Page,
  initial: Partial<AppState> = {},
): Promise<AppState> {
  const state = defaultState(initial);
  await page.route(STATE_ROUTE, async (route) => {
    if (route.request().method() !== "GET") return route.fallback();
    await route.fulfill({
      status: 200,
      contentType: "application/json; charset=utf-8",
      body: JSON.stringify(state),
    });
  });
  await page.goto("/");
  await page.getByText(state.toolMessage, { exact: false }).first().waitFor();
  return state;
}

// ---------------------------------------------------------------------------
// PATTERNS (copy into specs)
//
//  Load + select firmware, expect flash enabled:
//    const state = await gotoApp(page);
//    await page.setInputFiles("#firmware", firmwareFile());
//    await expect(page.locator("#flashButton")).toBeEnabled();
//
//  Mock a successful flash that bumps the L counter to 1:
//    await page.route(FLASH_ROUTE, (route) =>
//      route.fulfill({
//        status: 200,
//        contentType: "application/x-ndjson; charset=utf-8",
//        body: ndjson([
//          { level: "info", message: "J-Link 000680123456" },
//          { level: "success", message: "L 側燒錄成功", done: true, success: true,
//            state: defaultState({ leftCount: 1 }) },
//        ]),
//      }));
//    // NOTE: the flash handler pings GET /api/state BEFORE POSTing, so keep the
//    // gotoApp state's busy=false (default) or the button stays disabled.
//
//  Mock a rejected request (HTTP error path -> networkHint/appendLog):
//    await page.route(RESET_ROUTE, (route) =>
//      route.fulfill({ status: 409, contentType: "application/json",
//        body: JSON.stringify({ error: "燒錄進行中，無法清除計數" }) }));
//
//  Observe busy-lock during install (no pre-ping) with a gate:
//    const gate = makeGate();
//    await page.route(INSTALL_ROUTE, async (route) => {
//      await gate.opened;
//      await route.fulfill({ status: 200, contentType: "application/x-ndjson",
//        body: ndjson([{ level: "success", message: "device 命令安裝完成",
//          done: true, success: true, state: defaultState() }]) });
//    });
//    await page.locator("#installButton").click();
//    await expect(page.locator("#installButton")).toBeDisabled();
//    gate.open();
// ---------------------------------------------------------------------------
