import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  BACKGROUND_CUSTOM,
  BACKGROUND_PRESET,
  BACKGROUND_PRESETS,
  BACKGROUND_TRANSPARENT,
  CUSTOM_BACKGROUND_SCENE_PAD,
  CHROME_MINIMAL,
  CHROME_OS,
  OS_STYLE_MACOS,
  OS_STYLE_WINDOWS,
  OS_STYLE_WIREFRAME,
  COMPACT_SUPER_SAMPLE,
  defaultExportOptions,
  exportFilename,
  exportScales,
  themeChromeAccents,
  macosTerminalInset,
  titleBarHeight,
  validateExportOptions,
  viewerFrameMetrics,
} from "./export-options.js";
import { computeLayout } from "./export-compositor.js";

describe("export-options", () => {
  it("rejects invalid chrome preset", () => {
    const opts = defaultExportOptions();
    opts.chrome_preset = "native";
    assert.throws(() => validateExportOptions(opts), /chrome_preset/);
  });

  it("scale 2 doubles outer width", () => {
    const screen = { cols: 10, rows: 2, lines: ["a", "b"] };
    const a = computeLayout(screen, defaultExportOptions());
    const b = computeLayout(screen, { ...defaultExportOptions(), scale: 2 });
    assert.equal(b.outerW, a.outerW * 2);
  });

  it("1x export supersamples then downscales for readable accents", () => {
    const scales = exportScales(1, 12);
    assert.equal(scales.renderScale, COMPACT_SUPER_SAMPLE);
    assert.equal(scales.downscale, COMPACT_SUPER_SAMPLE);
    assert.equal(scales.fontPx, 12);
  });

  it("2x export renders at full resolution", () => {
    const scales = exportScales(2, 14);
    assert.equal(scales.renderScale, 2);
    assert.equal(scales.downscale, 1);
  });

  it("os wireframe title bar is taller than minimal", () => {
    assert.ok(titleBarHeight(CHROME_OS, OS_STYLE_WIREFRAME) > titleBarHeight(CHROME_MINIMAL));
  });

  it("normalizes legacy os-wireframe preset", () => {
    const opts = validateExportOptions({ ...defaultExportOptions(), chrome_preset: "os-wireframe" });
    assert.equal(opts.chrome_preset, CHROME_OS);
    assert.equal(opts.chrome_os_style, OS_STYLE_WIREFRAME);
  });

  it("macos os style validates", () => {
    const opts = validateExportOptions({
      ...defaultExportOptions(),
      chrome_preset: CHROME_OS,
      chrome_os_style: OS_STYLE_MACOS,
    });
    assert.equal(opts.chrome_os_style, OS_STYLE_MACOS);
  });

  it("windows os style validates", () => {
    const opts = validateExportOptions({
      ...defaultExportOptions(),
      chrome_preset: CHROME_OS,
      chrome_os_style: OS_STYLE_WINDOWS,
    });
    assert.equal(opts.chrome_os_style, OS_STYLE_WINDOWS);
  });

  it("windows layout insets terminal content like macOS", () => {
    const screen = { cols: 10, rows: 2, lines: ["a", "b"] };
    const windows = computeLayout(screen, {
      ...defaultExportOptions(),
      chrome_preset: CHROME_OS,
      chrome_os_style: OS_STYLE_WINDOWS,
    });
    assert.equal(windows.osStyle, OS_STYLE_WINDOWS);
    assert.equal(windows.termInset, 8 * windows.renderScale);
    assert.equal(windows.titleBar, 36 * windows.renderScale);
    assert.equal(windows.termY, windows.titleBar + windows.termInset);
    assert.equal(windows.termX, windows.termInset);
    assert.equal(windows.renderOuterW, windows.termW + windows.termInset * 2);
    assert.equal(windows.renderOuterH, windows.titleBar + windows.termH + windows.termInset * 2);
  });

  it("macos layout insets terminal content from window edge", () => {
    const screen = { cols: 10, rows: 2, lines: ["a", "b"] };
    const macos = computeLayout(screen, {
      ...defaultExportOptions(),
      chrome_preset: CHROME_OS,
      chrome_os_style: OS_STYLE_MACOS,
    });
    assert.equal(macos.termInset, macosTerminalInset() * macos.renderScale);
    assert.equal(macos.termX, macos.termInset);
    assert.equal(macos.termY, macos.titleBar + macos.termInset);
    assert.equal(macos.renderOuterW, macos.termW + macos.termInset * 2);
    assert.equal(macos.renderOuterH, macos.titleBar + macos.termH + macos.termInset * 2);
  });

  it("viewer frame metrics match observe mode", () => {
    const m = viewerFrameMetrics(2);
    assert.equal(m.framePad, 28);
    assert.equal(m.radius, 20);
  });

  it("theme accents vary by preset", () => {
    const sunset = themeChromeAccents("sunset");
    const slate = themeChromeAccents("slate");
    assert.notEqual(sunset.border, slate.border);
    assert.notEqual(sunset.labelText, slate.labelText);
  });

  it("background presets include slate", () => {
    assert.ok(BACKGROUND_PRESETS.slate);
  });

  it("accepts custom background mode", () => {
    const opts = validateExportOptions({
      ...defaultExportOptions(),
      background_mode: BACKGROUND_CUSTOM,
    });
    assert.equal(opts.background_mode, BACKGROUND_CUSTOM);
  });

  it("viewer frame metrics use terminal bg for custom background", () => {
    const custom = viewerFrameMetrics(1, {
      ...defaultExportOptions(),
      background_mode: BACKGROUND_CUSTOM,
    });
    const preset = viewerFrameMetrics(1, {
      ...defaultExportOptions(),
      background_mode: BACKGROUND_PRESET,
      background_preset: "slate",
    });
    assert.equal(custom.frameBg, custom.termBg);
    assert.notEqual(custom.frameBg, preset.frameBg);
  });

  it("transparent minimal chrome keeps opaque viewer frame fill", () => {
    const transparent = viewerFrameMetrics(1, {
      ...defaultExportOptions(),
      background_mode: BACKGROUND_TRANSPARENT,
    });
    const custom = viewerFrameMetrics(1, {
      ...defaultExportOptions(),
      background_mode: BACKGROUND_CUSTOM,
    });
    assert.notEqual(transparent.frameBg, custom.frameBg);
  });

  it("custom background expands os chrome layout with scene margin", () => {
    const screen = { cols: 10, rows: 2, lines: ["a", "b"] };
    const base = computeLayout(screen, {
      ...defaultExportOptions(),
      chrome_preset: CHROME_OS,
      chrome_os_style: OS_STYLE_MACOS,
    });
    const custom = computeLayout(screen, {
      ...defaultExportOptions(),
      chrome_preset: CHROME_OS,
      chrome_os_style: OS_STYLE_MACOS,
      background_mode: BACKGROUND_CUSTOM,
    });
    assert.ok(custom.outerW > base.outerW);
    assert.ok(custom.outerH > base.outerH);
    assert.equal(custom.scenePad, CUSTOM_BACKGROUND_SCENE_PAD * custom.renderScale);
  });

  it("exportFilename uses title and sanitizes unsafe chars", () => {
    assert.equal(exportFilename("My Demo", "png"), "My Demo.png");
    assert.equal(exportFilename('bad/name:test', "svg"), "badnametest.svg");
    assert.equal(exportFilename("  ", "png"), "tuile.png");
  });
});
