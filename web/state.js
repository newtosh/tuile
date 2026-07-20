function atom(initial) {
  let value = initial;
  const listeners = new Set();
  return {
    get() {
      return value;
    },
    set(next) {
      value = next;
      for (const listener of listeners) {
        listener(value);
      }
    },
    subscribe(listener) {
      listeners.add(listener);
      listener(value);
      return () => listeners.delete(listener);
    },
  };
}

export const sessions = atom([]);
export const activeSessionId = atom(null);
export const uiStatus = atom("Initializing…");
export const uiBadge = atom({ text: "connecting", className: "" });
