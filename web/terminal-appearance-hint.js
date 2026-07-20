/**
 * Heuristics for suggesting app appearance changes when rendered terminal
 * colors clash with the current chrome (e.g. Neovim dark theme in light UI).
 */

export const MIN_SAMPLED_CELLS = 120;
export const APPEARANCE_MISMATCH_RATIO = 0.55;
export const DARK_LUMINANCE_THRESHOLD = 0.45;

const THEME_ANSI_KEYS = [
  "black",
  "red",
  "green",
  "yellow",
  "blue",
  "magenta",
  "cyan",
  "white",
  "brightBlack",
  "brightRed",
  "brightGreen",
  "brightYellow",
  "brightBlue",
  "brightMagenta",
  "brightCyan",
  "brightWhite",
];

export function unpackRgb(packed) {
  return {
    r: (packed >> 16) & 255,
    g: (packed >> 8) & 255,
    b: packed & 255,
  };
}

export function hexToRgb(hex) {
  const normalized = hex.replace("#", "");
  if (normalized.length !== 6) {
    return null;
  }
  return unpackRgb(parseInt(normalized, 16));
}

export function relativeLuminance({ r, g, b }) {
  const channel = (value) => {
    const c = value / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  };
  const rs = channel(r);
  const gs = channel(g);
  const bs = channel(b);
  return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
}

export function isDarkRgb(rgb) {
  return relativeLuminance(rgb) < DARK_LUMINANCE_THRESHOLD;
}

function themePaletteRgb(term) {
  const theme = term?.options?.theme ?? {};
  return THEME_ANSI_KEYS.map((key) => hexToRgb(theme[key] ?? "#000000"));
}

function paletteRgb(term, index) {
  const ansi = term?._core?._themeService?.colors?.ansi;
  if (ansi?.[index]?.rgba != null) {
    return unpackRgb(ansi[index].rgba);
  }
  const fallback = themePaletteRgb(term);
  if (index < 16) {
    return fallback[index] ?? fallback[0];
  }
  return fallback[0];
}

function resolveCellColorRgb(term, cell, channel) {
  const isDefault = channel === "fg" ? cell.isFgDefault() : cell.isBgDefault();
  const isRgb = channel === "fg" ? cell.isFgRGB() : cell.isBgRGB();
  const isPalette = channel === "fg" ? cell.isFgPalette() : cell.isBgPalette();
  const raw = channel === "fg" ? cell.getFgColor() : cell.getBgColor();
  const theme = term?.options?.theme ?? {};

  if (isRgb) {
    return unpackRgb(raw);
  }
  if (isDefault) {
    const hex = channel === "fg" ? theme.foreground : theme.background;
    return hex ? hexToRgb(hex) : null;
  }
  if (isPalette) {
    return paletteRgb(term, raw);
  }
  return null;
}

export function resolveCellFillRgb(term, cell) {
  if (!cell) {
    return null;
  }
  if (cell.isInverse?.()) {
    return resolveCellColorRgb(term, cell, "fg");
  }
  return resolveCellColorRgb(term, cell, "bg");
}

export function analyzeTerminalBuffer(term) {
  const buffer = term?.buffer?.active;
  if (!buffer) {
    return { sampled: 0, dark: 0, light: 0, darkRatio: 0, lightRatio: 0 };
  }

  let dark = 0;
  let light = 0;
  let sampled = 0;

  for (let row = 0; row < buffer.length; row++) {
    const line = buffer.getLine(row);
    if (!line) {
      continue;
    }
    for (let col = 0; col < line.length; col++) {
      const cell = line.getCell(col);
      if (!cell) {
        continue;
      }
      const chars = cell.getChars() ?? "";
      const hasExplicitFill =
        cell.isInverse?.() || !cell.isBgDefault?.() || !cell.isFgDefault?.();
      if (!chars.trim() && !hasExplicitFill) {
        continue;
      }
      const rgb = resolveCellFillRgb(term, cell);
      if (!rgb) {
        continue;
      }
      sampled++;
      if (isDarkRgb(rgb)) {
        dark++;
      } else {
        light++;
      }
    }
  }

  return {
    sampled,
    dark,
    light,
    darkRatio: sampled ? dark / sampled : 0,
    lightRatio: sampled ? light / sampled : 0,
  };
}

export function shouldSuggestAppearanceSwitch(appAppearance, analysis) {
  if (analysis.sampled < MIN_SAMPLED_CELLS) {
    return null;
  }
  if (appAppearance === "light" && analysis.darkRatio >= APPEARANCE_MISMATCH_RATIO) {
    return "dark";
  }
  if (appAppearance === "dark" && analysis.lightRatio >= APPEARANCE_MISMATCH_RATIO) {
    return "light";
  }
  return null;
}

export function appearanceHintCopy(suggestion) {
  if (suggestion === "dark") {
    return {
      text: "Switch app appearance to Dark for a better frame match?",
      action: "Switch to dark",
    };
  }
  return {
    text: "Switch app appearance to Light for a better frame match?",
    action: "Switch to light",
  };
}
