import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  ACK_STORAGE_KEY,
  activeSessionIDSet,
  loadAckMap,
  mergeSessionsWithConnected,
  pruneAckMap,
  pruneClientSessionState,
  pruneSessionCache,
  saveAckMap,
} from "./session-state.js";

class MemoryStorage {
  constructor() {
    this.store = new Map();
  }

  getItem(key) {
    return this.store.has(key) ? this.store.get(key) : null;
  }

  setItem(key, value) {
    this.store.set(key, value);
  }
}

describe("session-state", () => {
  it("pruneSessionCache drops stale IDs", () => {
    const cache = new Map([
      ["a", { token: "1" }],
      ["b", { token: "2" }],
      ["c", { token: "3" }],
    ]);
    pruneSessionCache(cache, new Set(["a", "c"]));
    assert.deepEqual([...cache.keys()], ["a", "c"]);
  });

  it("pruneAckMap keeps only active session IDs", () => {
    const storage = new MemoryStorage();
    saveAckMap(
      {
        a: "2026-01-01T00:00:00.000Z",
        b: "2026-01-01T00:00:00.000Z",
        c: "2026-01-01T00:00:00.000Z",
      },
      storage
    );

    pruneAckMap(new Set(["b"]), storage);

    assert.deepEqual(loadAckMap(storage), { b: "2026-01-01T00:00:00.000Z" });
  });

  it("pruneClientSessionState orchestrates cache and ack pruning", () => {
    const storage = new MemoryStorage();
    const cache = new Map([
      ["keep", { token: "x" }],
      ["drop", { token: "y" }],
    ]);
    saveAckMap({ keep: "t1", drop: "t2" }, storage);

    const active = pruneClientSessionState({
      cache,
      sessions: [{ session_id: "keep" }],
      storage,
    });

    assert.equal(active.size, 1);
    assert.deepEqual([...cache.keys()], ["keep"]);
    assert.deepEqual(loadAckMap(storage), { keep: "t1" });
    assert.equal(ACK_STORAGE_KEY, "tuile_session_ack");
  });

  it("activeSessionIDSet ignores rows without session_id", () => {
    const ids = activeSessionIDSet([{ session_id: "ok" }, {}, { session_id: "" }]);
    assert.deepEqual([...ids], ["ok"]);
  });

  it("mergeSessionsWithConnected appends only when missing", () => {
    const connected = { session_id: "b", workspace: "/tmp" };
    const base = [{ session_id: "a" }];
    assert.deepEqual(mergeSessionsWithConnected(base, connected), [...base, connected]);
    assert.deepEqual(mergeSessionsWithConnected([...base, connected], connected), [...base, connected]);
    assert.deepEqual(mergeSessionsWithConnected(base, null), base);
  });
});
