import test from "node:test";
import assert from "node:assert/strict";
import { FALLBACK_LIGATURES, ligatureRanges } from "./ligatures.js";

test("ligatureRanges merges lua concatenation", () => {
  const ranges = ligatureRanges('"Hello, " .. name', FALLBACK_LIGATURES);
  assert.deepEqual(ranges, [[10, 12]]);
});

test("ligatureRanges prefers longer matches", () => {
  const ranges = ligatureRanges("a === b", FALLBACK_LIGATURES);
  assert.deepEqual(ranges, [[2, 5]]);
});
