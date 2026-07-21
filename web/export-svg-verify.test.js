import test from "node:test";
import assert from "node:assert/strict";
import {
  assertValidSvgRasterExport,
  parseSvgRasterExport,
  pngDimensionsFromBase64,
  validateSvgRasterExport,
} from "./export-svg-verify.js";

// 2x2 opaque RGB PNG
const RGB_2X2 =
  "iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAIAAAD91JpzAAAAEklEQVR4nGPk5uFnYGBgYgADAAHuACpswSxYAAAAAElFTkSuQmCC";

test("parse and validate aligned raster svg with png", () => {
  const svg = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="2" height="2" viewBox="0 0 2 2">
<image x="0" y="0" width="2" height="2" href="data:image/png;base64,${RGB_2X2}"/>
</svg>`;
  const parsed = assertValidSvgRasterExport(svg);
  assert.equal(parsed.width, 2);
  assert.equal(pngDimensionsFromBase64(parsed.rasterBase64).colorType, 2);
});

test("allows supersampled raster larger than svg viewport", () => {
  const svg = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1" viewBox="0 0 1 1">
<image x="0" y="0" width="1" height="1" href="data:image/png;base64,${RGB_2X2}"/>
</svg>`;
  const parsed = assertValidSvgRasterExport(svg);
  assert.equal(parsed.width, 1);
  assert.equal(pngDimensionsFromBase64(parsed.rasterBase64).width, 2);
});
test("rejects viewBox mismatch", () => {
  const svg = `<?xml version="1.0"?><svg width="1018" height="642" viewBox="0 0 2036 1284"><image x="0" y="0" width="1018" height="642" href="data:image/png;base64,${RGB_2X2}"/></svg>`;
  const errors = validateSvgRasterExport(parseSvgRasterExport(svg), svg);
  assert.ok(errors.some((e) => e.includes("viewBox")));
});

test("rejects nested scale transforms", () => {
  const svg = `<?xml version="1.0"?><svg width="2" height="2" viewBox="0 0 2 2"><g transform="scale(0.5)"><image x="0" y="0" width="2" height="2" href="data:image/png;base64,${RGB_2X2}"/></g></svg>`;
  const errors = validateSvgRasterExport(parseSvgRasterExport(svg), svg);
  assert.ok(errors.some((e) => e.includes("scale transforms")));
});
