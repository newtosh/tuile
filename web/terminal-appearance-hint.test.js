import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
  analyzeTerminalBuffer,
  appearanceHintCopy,
  hexToRgb,
  isDarkRgb,
  relativeLuminance,
  shouldSuggestAppearanceSwitch,
  unpackRgb,
} from "./terminal-appearance-hint.js";

describe("terminal-appearance-hint", () => {
  it("classifies luminance", () => {
    assert.ok(isDarkRgb(hexToRgb("#1e1e2e")));
    assert.ok(!isDarkRgb(hexToRgb("#f5f2ec")));
    assert.ok(relativeLuminance(hexToRgb("#ffffff")) > relativeLuminance(hexToRgb("#000000")));
  });

  it("unpacks rgb integers", () => {
    assert.deepEqual(unpackRgb(0xff8040), { r: 255, g: 128, b: 64 });
  });

  it("suggests dark app appearance for predominantly dark terminal output", () => {
    const suggestion = shouldSuggestAppearanceSwitch("light", {
      sampled: 500,
      dark: 420,
      light: 80,
      darkRatio: 0.84,
      lightRatio: 0.16,
    });
    assert.equal(suggestion, "dark");
  });

  it("suggests light app appearance for predominantly light terminal output", () => {
    const suggestion = shouldSuggestAppearanceSwitch("dark", {
      sampled: 500,
      dark: 60,
      light: 440,
      darkRatio: 0.12,
      lightRatio: 0.88,
    });
    assert.equal(suggestion, "light");
  });

  it("ignores low-confidence samples", () => {
    assert.equal(
      shouldSuggestAppearanceSwitch("light", {
        sampled: 40,
        dark: 40,
        light: 0,
        darkRatio: 1,
        lightRatio: 0,
      }),
      null,
    );
  });

  it("analyzes a mocked dark buffer", () => {
    const lines = Array.from({ length: 24 }, () =>
      Array.from({ length: 80 }, () => ({
        chars: "x",
        bg: 0x1e1e2e,
        bgMode: "rgb",
      })),
    );
    const term = {
      options: { theme: { background: "#f5f2ec", foreground: "#1c1b19" } },
      buffer: {
        active: {
          length: lines.length,
          getLine(y) {
            const row = lines[y];
            if (!row) {
              return null;
            }
            return {
              length: row.length,
              getCell(x) {
                const cell = row[x];
                if (!cell) {
                  return null;
                }
                return {
                  getChars: () => cell.chars,
                  isInverse: () => false,
                  isBgDefault: () => false,
                  isFgDefault: () => true,
                  isBgRGB: () => cell.bgMode === "rgb",
                  isBgPalette: () => false,
                  isFgRGB: () => false,
                  isFgPalette: () => false,
                  getBgColor: () => cell.bg,
                  getFgColor: () => 0,
                };
              },
            };
          },
        },
      },
    };
    const analysis = analyzeTerminalBuffer(term);
    assert.ok(analysis.sampled >= 120);
    assert.ok(analysis.darkRatio >= 0.9);
    assert.equal(shouldSuggestAppearanceSwitch("light", analysis), "dark");
  });

  it("provides hint copy for both directions", () => {
    assert.match(appearanceHintCopy("dark").text, /Switch app appearance to Dark/i);
    assert.match(appearanceHintCopy("dark").action, /dark/i);
    assert.match(appearanceHintCopy("light").text, /Switch app appearance to Light/i);
    assert.match(appearanceHintCopy("light").action, /light/i);
  });
});
