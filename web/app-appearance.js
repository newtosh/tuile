export function systemAppearance() {
  if (typeof window !== "undefined" && typeof window.matchMedia === "function") {
    return window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark";
  }
  return "dark";
}

export function normalizeAppAppearancePreference(stored) {
  if (stored === "light" || stored === "dark" || stored === "auto") {
    return stored;
  }
  return "dark";
}

export function resolveAppAppearance(preference) {
  const mode = normalizeAppAppearancePreference(preference);
  if (mode === "auto") {
    return systemAppearance();
  }
  return mode;
}
