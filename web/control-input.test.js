import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { encodeControlKey, isModifierKey, isPrintableKey } from "./control-input.js";

function keyEvent(overrides) {
  return {
    key: "",
    altKey: false,
    metaKey: false,
    ctrlKey: false,
    ...overrides,
  };
}

describe("encodeControlKey", () => {
  it("encodes printable characters including space", () => {
    assert.equal(encodeControlKey(keyEvent({ key: "t" })), "t");
    assert.equal(encodeControlKey(keyEvent({ key: " " })), " ");
  });

  it("encodes special keys", () => {
    assert.equal(encodeControlKey(keyEvent({ key: "Enter" })), "\r");
    assert.equal(encodeControlKey(keyEvent({ key: "Backspace" })), "\x7f");
    assert.equal(encodeControlKey(keyEvent({ key: "ArrowUp" })), "\x1b[A");
  });

  it("encodes emacs/shell ctrl sequences", () => {
    assert.equal(encodeControlKey(keyEvent({ key: "u", ctrlKey: true })), "\x15");
    assert.equal(encodeControlKey(keyEvent({ key: "c", ctrlKey: true })), "\x03");
    assert.equal(encodeControlKey(keyEvent({ key: "a", ctrlKey: true })), "\x01");
  });

  it("encodes alt+char as esc prefix", () => {
    assert.equal(encodeControlKey(keyEvent({ key: "b", altKey: true })), "\x1bb");
  });
});

describe("helpers", () => {
  it("detects printable keys", () => {
    assert.equal(isPrintableKey(keyEvent({ key: "a" })), true);
    assert.equal(isPrintableKey(keyEvent({ key: "Enter" })), false);
  });

  it("detects bare modifiers", () => {
    assert.equal(isModifierKey(keyEvent({ key: "Control" })), true);
  });
});
