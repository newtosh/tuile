#!/usr/bin/env node
/**
 * Headed browser test for control-mode keyboard input (no duplicate keystrokes).
 *
 * Usage:
 *   TUILE_CONTROL_URL='http://127.0.0.1:7710/view?session=...&token=...' \
 *     NODE_PATH=test/browser/node_modules node scripts/test-control-input-playwright.cjs
 *
 * Or: ./scripts/test-control-input.sh
 */

const { chromium } = require("playwright");
const fs = require("fs");
const path = require("path");

const rawUrl = process.env.TUILE_CONTROL_URL;
if (!rawUrl) {
  console.error("TUILE_CONTROL_URL is required (control viewer URL with session+token)");
  process.exit(2);
}

const url = (() => {
  const u = new URL(rawUrl);
  u.searchParams.set("test", "1");
  return u.toString();
})();

const headless = process.env.HEADLESS === "1";
const slowMs = Number(process.env.TUILE_TYPE_DELAY_MS || 40);
const screenshotPath =
  process.env.TUILE_SCREENSHOT || "test/integration/control-input-playwright.png";

function typedFromScreen(text) {
  const lines = text.split("\n").filter((l) => l.trim());
  for (let i = lines.length - 1; i >= 0; i--) {
    let line = lines[i].trimEnd();
    if (!line.trim()) continue;
    for (const sep of [">", "❯", "$", "#", "%"]) {
      const idx = line.lastIndexOf(sep);
      if (idx >= 0) {
        return line.slice(idx + 1).trim();
      }
    }
    return line.trim();
  }
  return "";
}

function hasAdjacentDupes(s) {
  for (let i = 0; i < s.length - 1; i++) {
    if (s[i] === s[i + 1] && s[i] !== "-") return true;
  }
  return false;
}

async function waitForTestHook(page) {
  await page.waitForFunction(
    () => typeof window.__tuileTest?.screenText === "function",
    null,
    { timeout: 45000 },
  );
}

async function waitForControl(page) {
  await page.waitForFunction(
    () =>
      window.__tuileTest?.isControlMode?.() &&
      !document.getElementById("release")?.disabled,
    null,
    { timeout: 20000 },
  );
}

async function readScreen(page) {
  return page.evaluate(() => window.__tuileTest.screenText());
}

async function waitForViewer(page) {
  await page.waitForFunction(
    () => !document.getElementById("takeover")?.disabled,
    null,
    { timeout: 30000 },
  );
}

async function enterControl(page) {
  await waitForViewer(page);
  await page.click("#settings-toggle");
  await page.waitForSelector("#takeover:not([disabled])", { visible: true, timeout: 10000 });
  await page.click("#takeover");
  await waitForControl(page);
  await page.click("#terminal-wrap");
  await page.waitForTimeout(300);
}

async function typeSlow(page, text) {
  for (const ch of text) {
    if (ch === "-") {
      await page.keyboard.press("Minus");
    } else if (ch === " ") {
      await page.keyboard.press("Space");
    } else {
      await page.keyboard.press(ch);
    }
    await page.waitForTimeout(slowMs);
  }
}

async function saveFailureShot(page, label) {
  const dir = path.dirname(screenshotPath);
  fs.mkdirSync(dir, { recursive: true });
  const failPath = screenshotPath.replace(/\.png$/, `-${label}.png`);
  await page.screenshot({ path: failPath, fullPage: true });
  console.error("Failure screenshot:", failPath);
}

async function main() {
  const launchOpts = {
    headless,
    args: ["--no-sandbox"],
  };
  if (process.env.CHROME_PATH) {
    launchOpts.executablePath = process.env.CHROME_PATH;
  }

  const browser = await chromium.launch(launchOpts);
  const page = await browser.newPage();
  page.setDefaultTimeout(30000);
  page.on("console", (msg) => {
    if (msg.type() === "error") {
      console.error("browser:", msg.text());
    }
  });

  try {
    console.log("Opening", url);
    await page.goto(url, { waitUntil: "domcontentloaded" });
    await waitForTestHook(page);
    await waitForViewer(page);

    await enterControl(page);

    const typed = "tuile-ok";
    console.log("Typing:", typed);
    await typeSlow(page, typed);
    await page.waitForTimeout(800);

    let screen = await readScreen(page);
    let line = typedFromScreen(screen);
    console.log("After tuile-ok:", JSON.stringify(line));

    if (line !== typed) {
      await saveFailureShot(page, "tuile-ok");
      throw new Error(
        `typed line mismatch: got ${JSON.stringify(line)} want ${JSON.stringify(typed)}\nscreen tail: ${screen.slice(-300)}`,
      );
    }
    if (hasAdjacentDupes(line) && line !== typed) {
      await saveFailureShot(page, "dupes");
      throw new Error(`adjacent duplicate runes in: ${JSON.stringify(line)}`);
    }

    await page.keyboard.press("Enter");
    await page.waitForTimeout(800);

    console.log("Typing: xy");
    await typeSlow(page, "xy");
    await page.waitForTimeout(600);
    screen = await readScreen(page);
    line = typedFromScreen(screen);
    console.log("After xy:", JSON.stringify(line));

    if (line !== "xy") {
      await saveFailureShot(page, "xy");
      throw new Error(`xy mismatch: got ${JSON.stringify(line)} want "xy"\nscreen tail: ${screen.slice(-300)}`);
    }

    fs.mkdirSync(path.dirname(screenshotPath), { recursive: true });
    await page.screenshot({ path: screenshotPath });
    console.log("Screenshot:", screenshotPath);
    console.log("PASS: control input has no obvious duplicate keystrokes");
  } finally {
    await browser.close();
  }
}

main().catch((err) => {
  console.error("FAIL:", err.message || err);
  process.exit(1);
});
