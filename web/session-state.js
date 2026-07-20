export const ACK_STORAGE_KEY = "tuile_session_ack";

export function loadAckMap(storage = localStorage, key = ACK_STORAGE_KEY) {
  try {
    const raw = storage.getItem(key);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

export function saveAckMap(map, storage = localStorage, key = ACK_STORAGE_KEY) {
  storage.setItem(key, JSON.stringify(map));
}

export function mergeSessionsWithConnected(sessions, connected) {
  if (!connected?.session_id) {
    return sessions;
  }
  const list = Array.isArray(sessions) ? sessions : [];
  if (list.some((sess) => sess.session_id === connected.session_id)) {
    return list;
  }
  return [...list, connected];
}

export function activeSessionIDSet(sessions) {
  const ids = new Set();
  for (const sess of sessions) {
    if (sess?.session_id) {
      ids.add(sess.session_id);
    }
  }
  return ids;
}

export function pruneSessionCache(cache, activeSessionIds) {
  for (const id of cache.keys()) {
    if (!activeSessionIds.has(id)) {
      cache.delete(id);
    }
  }
}

export function pruneAckMap(activeSessionIds, storage = localStorage, key = ACK_STORAGE_KEY) {
  const map = loadAckMap(storage, key);
  let changed = false;
  for (const id of Object.keys(map)) {
    if (!activeSessionIds.has(id)) {
      delete map[id];
      changed = true;
    }
  }
  if (changed) {
    saveAckMap(map, storage, key);
  }
  return map;
}

export function pruneClientSessionState({ cache, sessions, storage = localStorage }) {
  const activeSessionIds = activeSessionIDSet(sessions);
  pruneSessionCache(cache, activeSessionIds);
  pruneAckMap(activeSessionIds, storage);
  return activeSessionIds;
}
