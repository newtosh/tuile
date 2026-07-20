/**
 * Terminal ANSI theme registry for the Tuile browser viewer.
 *
 * Export contract (for feat/terminal-export merge):
 * - getTerminalTheme(id) -> { id, label, family, variant, appearance, xterm }
 * - listTerminalThemes() -> sorted entries for UI
 * - defaultTerminalThemeId
 *
 * Theme ids are stable `family:variant` strings.
 */

export const defaultTerminalThemeId = "tuile:default";
export const defaultLightTerminalThemeId = "tuile:light";

function theme(palette, meta) {
  return {
    ...meta,
    xterm: {
      cursor: palette.cursor ?? palette.foreground,
      cursorAccent: palette.cursorAccent ?? palette.background,
      selectionBackground:
        palette.selectionBackground ?? "rgba(121, 192, 255, 0.35)",
      ...palette,
    },
  };
}

/** @type {Record<string, ReturnType<typeof theme>>} */
export const TERMINAL_THEMES = {
  "tuile:default": theme(
    {
      background: "#0a0a0a",
      foreground: "#e4e4e4",
      cursor: "#f97316",
      cursorAccent: "#0a0a0a",
      selectionBackground: "rgba(121, 192, 255, 0.35)",
      black: "#0a0a0a",
      red: "#f87171",
      green: "#4ade80",
      yellow: "#facc15",
      blue: "#60a5fa",
      magenta: "#c084fc",
      cyan: "#22d3ee",
      white: "#e4e4e4",
      brightBlack: "#6b7280",
      brightRed: "#fca5a5",
      brightGreen: "#86efac",
      brightYellow: "#fde047",
      brightBlue: "#93c5fd",
      brightMagenta: "#d8b4fe",
      brightCyan: "#67e8f9",
      brightWhite: "#f9fafb",
    },
    {
      id: "tuile:default",
      label: "Tuile Default",
      family: "Tuile",
      variant: "default",
      appearance: "dark",
    },
  ),

  "tuile:light": theme(
    {
      background: "#f5f2ec",
      foreground: "#1c1b19",
      cursor: "#c47f1a",
      cursorAccent: "#f5f2ec",
      selectionBackground: "rgba(47, 127, 163, 0.28)",
      black: "#6f6c66",
      red: "#c44f4f",
      green: "#2f855a",
      yellow: "#9a7618",
      blue: "#2f7fa3",
      magenta: "#9b5de0",
      cyan: "#1a7f8f",
      white: "#1c1b19",
      brightBlack: "#8a8680",
      brightRed: "#b42323",
      brightGreen: "#1f7a45",
      brightYellow: "#7a5d10",
      brightBlue: "#256f91",
      brightMagenta: "#7c3aed",
      brightCyan: "#0f6f7d",
      brightWhite: "#0f0e0c",
    },
    {
      id: "tuile:light",
      label: "Tuile Light",
      family: "Tuile",
      variant: "light",
      appearance: "light",
    },
  ),

  "one-dark:dark": theme(
    {
      background: "#282c34",
      foreground: "#abb2bf",
      cursor: "#528bff",
      cursorAccent: "#282c34",
      black: "#282c34",
      red: "#e06c75",
      green: "#98c379",
      yellow: "#e5c07b",
      blue: "#61afef",
      magenta: "#c678dd",
      cyan: "#56b6c2",
      white: "#abb2bf",
      brightBlack: "#5c6370",
      brightRed: "#e06c75",
      brightGreen: "#98c379",
      brightYellow: "#e5c07b",
      brightBlue: "#61afef",
      brightMagenta: "#c678dd",
      brightCyan: "#56b6c2",
      brightWhite: "#ffffff",
    },
    {
      id: "one-dark:dark",
      label: "One Dark",
      family: "One Dark",
      variant: "dark",
      appearance: "dark",
    },
  ),

  "dracula:dark": theme(
    {
      background: "#282a36",
      foreground: "#f8f8f2",
      cursor: "#f8f8f0",
      cursorAccent: "#282a36",
      black: "#21222c",
      red: "#ff5555",
      green: "#50fa7b",
      yellow: "#f1fa8c",
      blue: "#bd93f9",
      magenta: "#ff79c6",
      cyan: "#8be9fd",
      white: "#f8f8f2",
      brightBlack: "#6272a4",
      brightRed: "#ff6e6e",
      brightGreen: "#69ff94",
      brightYellow: "#ffffa5",
      brightBlue: "#d6acff",
      brightMagenta: "#ff92df",
      brightCyan: "#a4ffff",
      brightWhite: "#ffffff",
    },
    {
      id: "dracula:dark",
      label: "Dracula",
      family: "Dracula",
      variant: "dark",
      appearance: "dark",
    },
  ),

  "catppuccin:mocha": theme(
    {
      background: "#1e1e2e",
      foreground: "#cdd6f4",
      cursor: "#f5e0dc",
      cursorAccent: "#1e1e2e",
      black: "#45475a",
      red: "#f38ba8",
      green: "#a6e3a1",
      yellow: "#f9e2af",
      blue: "#89b4fa",
      magenta: "#f5c2e7",
      cyan: "#94e2d5",
      white: "#bac2de",
      brightBlack: "#585b70",
      brightRed: "#f38ba8",
      brightGreen: "#a6e3a1",
      brightYellow: "#f9e2af",
      brightBlue: "#89b4fa",
      brightMagenta: "#f5c2e7",
      brightCyan: "#94e2d5",
      brightWhite: "#a6adc8",
    },
    {
      id: "catppuccin:mocha",
      label: "Catppuccin Mocha",
      family: "Catppuccin",
      variant: "mocha",
      appearance: "dark",
    },
  ),

  "catppuccin:macchiato": theme(
    {
      background: "#24273a",
      foreground: "#cad3f5",
      cursor: "#f4dbd6",
      cursorAccent: "#24273a",
      black: "#494d64",
      red: "#ed8796",
      green: "#a6da95",
      yellow: "#eed49f",
      blue: "#8aadf4",
      magenta: "#f5bde6",
      cyan: "#8bd5ca",
      white: "#b8c0e0",
      brightBlack: "#5b6078",
      brightRed: "#ed8796",
      brightGreen: "#a6da95",
      brightYellow: "#eed49f",
      brightBlue: "#8aadf4",
      brightMagenta: "#f5bde6",
      brightCyan: "#8bd5ca",
      brightWhite: "#a5adcb",
    },
    {
      id: "catppuccin:macchiato",
      label: "Catppuccin Macchiato",
      family: "Catppuccin",
      variant: "macchiato",
      appearance: "dark",
    },
  ),

  "catppuccin:frappe": theme(
    {
      background: "#303446",
      foreground: "#c6d0f5",
      cursor: "#f2d5cf",
      cursorAccent: "#303446",
      black: "#51576d",
      red: "#e78284",
      green: "#a6d189",
      yellow: "#e5c890",
      blue: "#8caaee",
      magenta: "#f4b8e4",
      cyan: "#81c8be",
      white: "#b5bfe2",
      brightBlack: "#626880",
      brightRed: "#e78284",
      brightGreen: "#a6d189",
      brightYellow: "#e5c890",
      brightBlue: "#8caaee",
      brightMagenta: "#f4b8e4",
      brightCyan: "#81c8be",
      brightWhite: "#a5adce",
    },
    {
      id: "catppuccin:frappe",
      label: "Catppuccin Frappé",
      family: "Catppuccin",
      variant: "frappe",
      appearance: "dark",
    },
  ),

  "catppuccin:latte": theme(
    {
      background: "#eff1f5",
      foreground: "#4c4f69",
      cursor: "#dc8a78",
      cursorAccent: "#eff1f5",
      black: "#5c5f77",
      red: "#d20f39",
      green: "#40a02b",
      yellow: "#df8e1d",
      blue: "#1e66f5",
      magenta: "#ea76cb",
      cyan: "#179299",
      white: "#acb0be",
      brightBlack: "#6c6f85",
      brightRed: "#d20f39",
      brightGreen: "#40a02b",
      brightYellow: "#df8e1d",
      brightBlue: "#1e66f5",
      brightMagenta: "#ea76cb",
      brightCyan: "#179299",
      brightWhite: "#bcc0cc",
    },
    {
      id: "catppuccin:latte",
      label: "Catppuccin Latte",
      family: "Catppuccin",
      variant: "latte",
      appearance: "light",
    },
  ),

  "gruvbox:dark": theme(
    {
      background: "#282828",
      foreground: "#ebdbb2",
      cursor: "#ebdbb2",
      cursorAccent: "#282828",
      black: "#282828",
      red: "#cc241d",
      green: "#98971a",
      yellow: "#d79921",
      blue: "#458588",
      magenta: "#b16286",
      cyan: "#689d6a",
      white: "#a89984",
      brightBlack: "#928374",
      brightRed: "#fb4934",
      brightGreen: "#b8bb26",
      brightYellow: "#fabd2f",
      brightBlue: "#83a598",
      brightMagenta: "#d3869b",
      brightCyan: "#8ec07c",
      brightWhite: "#ebdbb2",
    },
    {
      id: "gruvbox:dark",
      label: "Gruvbox Dark",
      family: "Gruvbox",
      variant: "dark",
      appearance: "dark",
    },
  ),

  "gruvbox:light": theme(
    {
      background: "#fbf1c7",
      foreground: "#3c3836",
      cursor: "#3c3836",
      cursorAccent: "#fbf1c7",
      black: "#fbf1c7",
      red: "#cc241d",
      green: "#98971a",
      yellow: "#d79921",
      blue: "#458588",
      magenta: "#b16286",
      cyan: "#689d6a",
      white: "#7c6f64",
      brightBlack: "#928374",
      brightRed: "#9d0006",
      brightGreen: "#79740e",
      brightYellow: "#b57614",
      brightBlue: "#076678",
      brightMagenta: "#8f3f71",
      brightCyan: "#427b58",
      brightWhite: "#3c3836",
    },
    {
      id: "gruvbox:light",
      label: "Gruvbox Light",
      family: "Gruvbox",
      variant: "light",
      appearance: "light",
    },
  ),

  "solarized:dark": theme(
    {
      background: "#002b36",
      foreground: "#839496",
      cursor: "#839496",
      cursorAccent: "#002b36",
      black: "#073642",
      red: "#dc322f",
      green: "#859900",
      yellow: "#b58900",
      blue: "#268bd2",
      magenta: "#d33682",
      cyan: "#2aa198",
      white: "#eee8d5",
      brightBlack: "#586e75",
      brightRed: "#cb4b16",
      brightGreen: "#586e75",
      brightYellow: "#657b83",
      brightBlue: "#839496",
      brightMagenta: "#6c71c4",
      brightCyan: "#93a1a1",
      brightWhite: "#fdf6e3",
    },
    {
      id: "solarized:dark",
      label: "Solarized Dark",
      family: "Solarized",
      variant: "dark",
      appearance: "dark",
    },
  ),

  "solarized:light": theme(
    {
      background: "#fdf6e3",
      foreground: "#657b83",
      cursor: "#657b83",
      cursorAccent: "#fdf6e3",
      black: "#073642",
      red: "#dc322f",
      green: "#859900",
      yellow: "#b58900",
      blue: "#268bd2",
      magenta: "#d33682",
      cyan: "#2aa198",
      white: "#eee8d5",
      brightBlack: "#586e75",
      brightRed: "#cb4b16",
      brightGreen: "#586e75",
      brightYellow: "#657b83",
      brightBlue: "#839496",
      brightMagenta: "#6c71c4",
      brightCyan: "#93a1a1",
      brightWhite: "#fdf6e3",
    },
    {
      id: "solarized:light",
      label: "Solarized Light",
      family: "Solarized",
      variant: "light",
      appearance: "light",
    },
  ),

  "tokyo-night:night": theme(
    {
      background: "#1a1b26",
      foreground: "#a9b1d6",
      cursor: "#c0caf5",
      cursorAccent: "#1a1b26",
      black: "#15161e",
      red: "#f7768e",
      green: "#9ece6a",
      yellow: "#e0af68",
      blue: "#7aa2f7",
      magenta: "#bb9af7",
      cyan: "#7dcfff",
      white: "#a9b1d6",
      brightBlack: "#414868",
      brightRed: "#f7768e",
      brightGreen: "#9ece6a",
      brightYellow: "#e0af68",
      brightBlue: "#7aa2f7",
      brightMagenta: "#bb9af7",
      brightCyan: "#7dcfff",
      brightWhite: "#c0caf5",
    },
    {
      id: "tokyo-night:night",
      label: "Tokyo Night",
      family: "Tokyo Night",
      variant: "night",
      appearance: "dark",
    },
  ),

  "tokyo-night:storm": theme(
    {
      background: "#24283b",
      foreground: "#c0caf5",
      cursor: "#c0caf5",
      cursorAccent: "#24283b",
      black: "#1d202f",
      red: "#f7768e",
      green: "#9ece6a",
      yellow: "#e0af68",
      blue: "#7aa2f7",
      magenta: "#bb9af7",
      cyan: "#7dcfff",
      white: "#a9b1d6",
      brightBlack: "#414868",
      brightRed: "#f7768e",
      brightGreen: "#9ece6a",
      brightYellow: "#e0af68",
      brightBlue: "#7aa2f7",
      brightMagenta: "#bb9af7",
      brightCyan: "#7dcfff",
      brightWhite: "#c0caf5",
    },
    {
      id: "tokyo-night:storm",
      label: "Tokyo Night Storm",
      family: "Tokyo Night",
      variant: "storm",
      appearance: "dark",
    },
  ),

  "tokyo-night:day": theme(
    {
      background: "#e1e2e7",
      foreground: "#3760bf",
      cursor: "#3760bf",
      cursorAccent: "#e1e2e7",
      black: "#e9e9ed",
      red: "#f52a65",
      green: "#587539",
      yellow: "#8c6c3e",
      blue: "#2e7de9",
      magenta: "#9854f1",
      cyan: "#007197",
      white: "#6172b0",
      brightBlack: "#a1a6c5",
      brightRed: "#f52a65",
      brightGreen: "#587539",
      brightYellow: "#8c6c3e",
      brightBlue: "#2e7de9",
      brightMagenta: "#9854f1",
      brightCyan: "#007197",
      brightWhite: "#3760bf",
    },
    {
      id: "tokyo-night:day",
      label: "Tokyo Night Day",
      family: "Tokyo Night",
      variant: "day",
      appearance: "light",
    },
  ),

  "rose-pine:main": theme(
    {
      background: "#191724",
      foreground: "#e0def4",
      cursor: "#e0def4",
      cursorAccent: "#191724",
      black: "#26233a",
      red: "#eb6f92",
      green: "#31748f",
      yellow: "#f6c177",
      blue: "#9ccfd8",
      magenta: "#c4a7e7",
      cyan: "#ebbcba",
      white: "#e0def4",
      brightBlack: "#6e6a86",
      brightRed: "#eb6f92",
      brightGreen: "#31748f",
      brightYellow: "#f6c177",
      brightBlue: "#9ccfd8",
      brightMagenta: "#c4a7e7",
      brightCyan: "#ebbcba",
      brightWhite: "#e0def4",
    },
    {
      id: "rose-pine:main",
      label: "Rosé Pine",
      family: "Rose Pine",
      variant: "main",
      appearance: "dark",
    },
  ),

  "rose-pine:moon": theme(
    {
      background: "#232136",
      foreground: "#e0def4",
      cursor: "#e0def4",
      cursorAccent: "#232136",
      black: "#393552",
      red: "#eb6f92",
      green: "#3e8fb0",
      yellow: "#f6c177",
      blue: "#9ccfd8",
      magenta: "#c4a7e7",
      cyan: "#ea9a97",
      white: "#e0def4",
      brightBlack: "#6e6a86",
      brightRed: "#eb6f92",
      brightGreen: "#3e8fb0",
      brightYellow: "#f6c177",
      brightBlue: "#9ccfd8",
      brightMagenta: "#c4a7e7",
      brightCyan: "#ea9a97",
      brightWhite: "#e0def4",
    },
    {
      id: "rose-pine:moon",
      label: "Rosé Pine Moon",
      family: "Rose Pine",
      variant: "moon",
      appearance: "dark",
    },
  ),

  "rose-pine:dawn": theme(
    {
      background: "#faf4ed",
      foreground: "#575279",
      cursor: "#575279",
      cursorAccent: "#faf4ed",
      black: "#f2e9e1",
      red: "#b4637a",
      green: "#286983",
      yellow: "#ea9d34",
      blue: "#56949f",
      magenta: "#907aa9",
      cyan: "#d7827e",
      white: "#575279",
      brightBlack: "#9893a5",
      brightRed: "#b4637a",
      brightGreen: "#286983",
      brightYellow: "#ea9d34",
      brightBlue: "#56949f",
      brightMagenta: "#907aa9",
      brightCyan: "#d7827e",
      brightWhite: "#575279",
    },
    {
      id: "rose-pine:dawn",
      label: "Rosé Pine Dawn",
      family: "Rose Pine",
      variant: "dawn",
      appearance: "light",
    },
  ),

  "github:dark": theme(
    {
      background: "#0d1117",
      foreground: "#c9d1d9",
      cursor: "#c9d1d9",
      cursorAccent: "#0d1117",
      black: "#484f58",
      red: "#ff7b72",
      green: "#3fb950",
      yellow: "#d29922",
      blue: "#58a6ff",
      magenta: "#bc8cff",
      cyan: "#39c5cf",
      white: "#b1bac4",
      brightBlack: "#6e7681",
      brightRed: "#ffa198",
      brightGreen: "#56d364",
      brightYellow: "#e3b341",
      brightBlue: "#79c0ff",
      brightMagenta: "#d2a8ff",
      brightCyan: "#56d4dd",
      brightWhite: "#f0f6fc",
    },
    {
      id: "github:dark",
      label: "GitHub Dark",
      family: "GitHub",
      variant: "dark",
      appearance: "dark",
    },
  ),

  "github:light": theme(
    {
      background: "#ffffff",
      foreground: "#24292f",
      cursor: "#24292f",
      cursorAccent: "#ffffff",
      black: "#24292f",
      red: "#cf222e",
      green: "#116329",
      yellow: "#4d2d00",
      blue: "#0969da",
      magenta: "#8250df",
      cyan: "#1b7c83",
      white: "#6e7781",
      brightBlack: "#57606a",
      brightRed: "#a40e26",
      brightGreen: "#1a7f37",
      brightYellow: "#633c01",
      brightBlue: "#218bff",
      brightMagenta: "#a475f9",
      brightCyan: "#3192aa",
      brightWhite: "#8c959f",
    },
    {
      id: "github:light",
      label: "GitHub Light",
      family: "GitHub",
      variant: "light",
      appearance: "light",
    },
  ),
};

export function listTerminalThemes() {
  return Object.values(TERMINAL_THEMES)
    .map(({ id, label, family, variant, appearance }) => ({
      id,
      label,
      family,
      variant,
      appearance,
    }))
    .sort((a, b) => a.label.localeCompare(b.label));
}

export function listTerminalThemesForAppearance(appearance) {
  const mode = appearance === "light" ? "light" : "dark";
  return listTerminalThemes().filter((entry) => entry.appearance === mode);
}

export function defaultTerminalThemeIdForAppearance(appearance) {
  return appearance === "light" ? defaultLightTerminalThemeId : defaultTerminalThemeId;
}

export function resolveTerminalThemeId(themeId, appearance) {
  const mode = appearance === "light" ? "light" : "dark";
  try {
    const entry = getTerminalTheme(themeId);
    if (entry.appearance === mode) {
      return themeId;
    }
  } catch {
    // fall through
  }
  return defaultTerminalThemeIdForAppearance(mode);
}

export function getTerminalTheme(id) {
  const entry = TERMINAL_THEMES[id];
  if (!entry) {
    throw new Error(`unknown terminal theme: ${id}`);
  }
  return entry;
}

export function getActiveTerminalTheme() {
  return getTerminalTheme(defaultTerminalThemeId);
}
