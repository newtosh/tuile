import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
  TERMINAL_THEMES,
  defaultLightTerminalThemeId,
  defaultTerminalThemeId,
  getTerminalTheme,
  listTerminalThemes,
  listTerminalThemesForAppearance,
  resolveTerminalThemeId,
} from "./terminal-themes.js";

const ANSI_KEYS = [
  "black",
  "red",
  "green",
  "yellow",
  "blue",
  "magenta",
  "cyan",
  "white",
  "brightBlack",
  "brightRed",
  "brightGreen",
  "brightYellow",
  "brightBlue",
  "brightMagenta",
  "brightCyan",
  "brightWhite",
];

const REQUIRED_KEYS = ["background", "foreground", ...ANSI_KEYS];

describe("terminal-themes", () => {
  it("default dark theme id resolves", () => {
    const theme = getTerminalTheme(defaultTerminalThemeId);
    assert.equal(theme.id, defaultTerminalThemeId);
    assert.equal(theme.appearance, "dark");
    assert.ok(theme.xterm.background);
  });

  it("default light theme id resolves", () => {
    const theme = getTerminalTheme(defaultLightTerminalThemeId);
    assert.equal(theme.id, "tuile:light");
    assert.equal(theme.appearance, "light");
    assert.equal(theme.xterm.background, "#f5f2ec");
  });

  it("every theme has unique id and valid ITheme shape", () => {
    const ids = new Set();
    for (const entry of listTerminalThemes()) {
      assert.ok(!ids.has(entry.id), `duplicate id ${entry.id}`);
      ids.add(entry.id);
      const theme = getTerminalTheme(entry.id);
      for (const key of REQUIRED_KEYS) {
        assert.ok(theme.xterm[key], `${entry.id} missing ${key}`);
      }
      assert.match(theme.xterm.background, /^#/);
      assert.match(theme.xterm.foreground, /^#/);
    }
  });

  it("catalog includes required families", () => {
    const families = new Set(listTerminalThemes().map((t) => t.family));
    for (const name of [
      "Tuile",
      "One Dark",
      "Dracula",
      "Catppuccin",
      "Gruvbox",
      "Solarized",
      "Tokyo Night",
      "Rose Pine",
      "GitHub",
    ]) {
      assert.ok(families.has(name), `missing family ${name}`);
    }
    assert.ok(listTerminalThemes().length >= 20);
  });

  it("filters themes by appearance", () => {
    const light = listTerminalThemesForAppearance("light");
    const dark = listTerminalThemesForAppearance("dark");
    assert.ok(light.length > 0);
    assert.ok(dark.length > 0);
    assert.ok(light.every((t) => t.appearance === "light"));
    assert.ok(dark.every((t) => t.appearance === "dark"));
    assert.ok(light.length + dark.length === listTerminalThemes().length);
  });

  it("resolves theme id for appearance", () => {
    assert.equal(resolveTerminalThemeId("dracula:dark", "light"), defaultLightTerminalThemeId);
    assert.equal(resolveTerminalThemeId("solarized:light", "dark"), defaultTerminalThemeId);
    assert.equal(resolveTerminalThemeId("solarized:light", "light"), "solarized:light");
  });

  it("getTerminalTheme throws for unknown id", () => {
    assert.throws(() => getTerminalTheme("nope:missing"), /unknown terminal theme/i);
  });

  it("registry object matches list", () => {
    assert.equal(Object.keys(TERMINAL_THEMES).length, listTerminalThemes().length);
  });
});
