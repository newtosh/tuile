import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

function sessionListBlock(css) {
  const match = css.match(/\.session-list\s*\{([^}]+)\}/);
  assert.ok(match, ".session-list rule not found in style.css");
  return match[1];
}

describe("style.css session sidebar", () => {
  const css = readFileSync(new URL("./style.css", import.meta.url), "utf8");
  const block = sessionListBlock(css);

  it("session list fills remaining panel height", () => {
    assert.match(block, /flex\s*:\s*1/);
  });

  it("session list can shrink inside flex column", () => {
    assert.match(block, /min-height\s*:\s*0/);
  });

  it("session list scrolls overflow instead of clipping", () => {
    assert.match(block, /overflow-y\s*:\s*auto/);
  });
});
