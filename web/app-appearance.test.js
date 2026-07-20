import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
  normalizeAppAppearancePreference,
  resolveAppAppearance,
  systemAppearance,
} from "./app-appearance.js";

describe("app-appearance", () => {
  it("normalizes stored preferences", () => {
    assert.equal(normalizeAppAppearancePreference("auto"), "auto");
    assert.equal(normalizeAppAppearancePreference("light"), "light");
    assert.equal(normalizeAppAppearancePreference("dark"), "dark");
    assert.equal(normalizeAppAppearancePreference("nope"), "dark");
    assert.equal(normalizeAppAppearancePreference(null), "dark");
  });

  it("resolves explicit preferences", () => {
    assert.equal(resolveAppAppearance("light"), "light");
    assert.equal(resolveAppAppearance("dark"), "dark");
  });

  it("resolves auto from system appearance", () => {
    const original = globalThis.window;
    globalThis.window = {
      matchMedia(query) {
        return { matches: query === "(prefers-color-scheme: light)" };
      },
    };
    try {
      assert.equal(systemAppearance(), "light");
      assert.equal(resolveAppAppearance("auto"), "light");
    } finally {
      globalThis.window = original;
    }
  });
});
