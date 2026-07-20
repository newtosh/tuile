import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  BACKGROUND_PRESETS,
  CHROME_MINIMAL,
  COMPACT_SUPER_SAMPLE,
  defaultExportOptions,
  exportFilename,
  exportScales,
  themeChromeAccents,
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

  it("os-wireframe title bar is taller", () => {
    assert.ok(titleBarHeight("os-wireframe") > titleBarHeight(CHROME_MINIMAL));
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

  it("exportFilename uses title and sanitizes unsafe chars", () => {
    assert.equal(exportFilename("My Demo", "png"), "My Demo.png");
    assert.equal(exportFilename('bad/name:test', "svg"), "badnametest.svg");
    assert.equal(exportFilename("  ", "png"), "tuile.png");
  });
});
