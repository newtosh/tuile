import { getTerminalTheme, resolveTerminalThemeId } from "./terminal-themes.js";

export const CHROME_MINIMAL = "minimal";
export const CHROME_OS = "os";
/** @deprecated use chrome_preset=os + chrome_os_style=wireframe */
export const CHROME_OS_WIREFRAME = "os-wireframe";

export const OS_STYLE_WIREFRAME = "wireframe";
export const OS_STYLE_MACOS = "macos";
export const OS_STYLE_WINDOWS = "windows";

export const BACKGROUND_TRANSPARENT = "transparent";
export const BACKGROUND_PRESET = "preset";
export const BACKGROUND_CUSTOM = "custom";

/** Visible wallpaper margin around chrome when exporting a custom background (1x px). */
export const CUSTOM_BACKGROUND_SCENE_PAD = 48;

export const FORMAT_PNG = "png";
export const FORMAT_SVG = "svg";

export const BACKGROUND_PRESETS = {
  slate: { kind: "solid", color: "#1e293b" },
  ink: { kind: "solid", color: "#0d1117" },
  mist: { kind: "solid", color: "#e2e8f0" },
  white: { kind: "solid", color: "#ffffff" },
  sunset: { kind: "gradient", start: "#f97316", end: "#7c3aed" },
  ocean: { kind: "gradient", start: "#0ea5e9", end: "#1e3a8a" },
  forest: { kind: "gradient", start: "#22c55e", end: "#14532d" },
  midnight: { kind: "gradient", start: "#1e1b4b", end: "#0f172a" },
};

// Viewer frame border + grid badge accents paired with each export theme.
export const THEME_CHROME_ACCENTS = {
  slate: {
    border: "#334e60",
    frameBg: "#0b0c0f",
    labelText: "#8ec8e0",
    labelBg: "rgba(22, 22, 28, 0.88)",
    labelBorder: "rgba(51, 78, 96, 0.55)",
  },
  ink: {
    border: "#3d444d",
    frameBg: "#0b0c0f",
    labelText: "#79c0ff",
    labelBg: "rgba(13, 17, 23, 0.92)",
    labelBorder: "rgba(88, 166, 255, 0.45)",
  },
  mist: {
    border: "#64748b",
    frameBg: "#0b0c0f",
    labelText: "#cbd5e1",
    labelBg: "rgba(30, 41, 59, 0.88)",
    labelBorder: "rgba(100, 116, 139, 0.5)",
  },
  white: {
    border: "#94a3b8",
    frameBg: "#0b0c0f",
    labelText: "#e2e8f0",
    labelBg: "rgba(30, 41, 59, 0.9)",
    labelBorder: "rgba(148, 163, 184, 0.45)",
  },
  sunset: {
    border: "#ea580c",
    frameBg: "#0b0c0f",
    labelText: "#fdba74",
    labelBg: "rgba(28, 15, 30, 0.9)",
    labelBorder: "rgba(249, 115, 22, 0.5)",
  },
  ocean: {
    border: "#0284c7",
    frameBg: "#0b0c0f",
    labelText: "#7dd3fc",
    labelBg: "rgba(12, 20, 40, 0.9)",
    labelBorder: "rgba(14, 165, 233, 0.45)",
  },
  forest: {
    border: "#15803d",
    frameBg: "#0b0c0f",
    labelText: "#86efac",
    labelBg: "rgba(10, 28, 18, 0.9)",
    labelBorder: "rgba(34, 197, 94, 0.45)",
  },
  midnight: {
    border: "#4f46e5",
    frameBg: "#0b0c0f",
    labelText: "#a5b4fc",
    labelBg: "rgba(15, 14, 35, 0.9)",
    labelBorder: "rgba(99, 102, 241, 0.45)",
  },
};

// Matches web/style.css observe-mode terminal frame (GRID_FRAME_PAD in app.js).
export const VIEWER_FRAME = {
  border: THEME_CHROME_ACCENTS.slate.border,
  frameBg: THEME_CHROME_ACCENTS.slate.frameBg,
  labelText: THEME_CHROME_ACCENTS.slate.labelText,
  labelBg: THEME_CHROME_ACCENTS.slate.labelBg,
  labelBorder: THEME_CHROME_ACCENTS.slate.labelBorder,
  framePad: 14,
  radius: 10,
};

// Matches .grid-frame-label in style.css (0.65rem @ 16px root).
export const GRID_LABEL = {
  fontRem: 0.65,
  rootPx: 16,
  padX: 6,
  padY: 2,
  offsetX: 6,
  offsetY: 5,
  radius: 4,
};

export function normalizeChromeOptions(opts) {
  const o = { ...opts };
  if (o.chrome_preset === CHROME_OS_WIREFRAME) {
    o.chrome_preset = CHROME_OS;
    o.chrome_os_style = o.chrome_os_style || OS_STYLE_WIREFRAME;
  }
  if (o.chrome_preset === CHROME_OS && !o.chrome_os_style) {
    o.chrome_os_style = OS_STYLE_WIREFRAME;
  }
  return o;
}

export function isOsChrome(opts) {
  const o = normalizeChromeOptions(opts);
  return o.chrome_preset === CHROME_OS;
}

export function resolveOsStyle(opts) {
  return normalizeChromeOptions(opts).chrome_os_style || OS_STYLE_WIREFRAME;
}

export function macosTitleBarHeight() {
  return MACOS_CHROME.titleBarHeight;
}

export function macosWindowRadius() {
  return MACOS_CHROME.windowRadius;
}

export const MACOS_CHROME = {
  titleBarHeight: 28,
  windowRadius: 10,
  terminalInset: 8,
  trafficLightSize: 12,
  trafficLightInsetX: 8,
  trafficLightInsetY: 8,
  trafficLightGap: 8,
  trafficLightColors: ["#F96057", "#F8CE52", "#5FCF65"],
  trafficLightRing: "rgba(0, 0, 0, 0.1)",
  titleFontSize: 13,
  shadowBlur: 30,
  shadowOffsetY: 20,
  shadowColor: "rgba(0, 0, 0, 0.2)",
  borderDark: "rgba(0, 0, 0, 0.35)",
  borderLight: "rgba(0, 0, 0, 0.12)",
  titleTextDark: "rgba(245, 245, 247, 0.72)",
  titleTextLight: "rgba(60, 60, 67, 0.72)",
  windowBgDark: "#0a0a0a",
  windowBgLight: "#ffffff",
};

export function macosChromePalette(opts, renderScale = 1) {
  const frame = viewerFrameMetrics(renderScale, opts);
  const light = opts?.theme === "light";
  const windowBg = frame.termBg || (light ? MACOS_CHROME.windowBgLight : MACOS_CHROME.windowBgDark);
  return {
    windowBg,
    titleBarBg: windowBg,
    border: light ? MACOS_CHROME.borderLight : MACOS_CHROME.borderDark,
    titleColor: light ? MACOS_CHROME.titleTextLight : MACOS_CHROME.titleTextDark,
    trafficLights: MACOS_CHROME.trafficLightColors,
    trafficRing: MACOS_CHROME.trafficLightRing,
    titleFontSize: MACOS_CHROME.titleFontSize * renderScale,
    titleBarHeight: MACOS_CHROME.titleBarHeight * renderScale,
    windowRadius: MACOS_CHROME.windowRadius * renderScale,
    trafficLightSize: MACOS_CHROME.trafficLightSize * renderScale,
    trafficLightInsetX: MACOS_CHROME.trafficLightInsetX * renderScale,
    trafficLightInsetY: MACOS_CHROME.trafficLightInsetY * renderScale,
    trafficLightGap: MACOS_CHROME.trafficLightGap * renderScale,
    shadowBlur: MACOS_CHROME.shadowBlur * renderScale,
    shadowOffsetY: MACOS_CHROME.shadowOffsetY * renderScale,
    shadowColor: MACOS_CHROME.shadowColor,
    terminalInset: MACOS_CHROME.terminalInset * renderScale,
  };
}

export function macosTerminalInset() {
  return MACOS_CHROME.terminalInset;
}

/** @deprecated macOS uses terminalInset instead. */
export function macosContentPadding() {
  return macosTerminalInset();
}

export function windowsTitleBarHeight() {
  return WINDOWS_CHROME.tabRowHeight;
}

export function windowsWindowRadius() {
  return WINDOWS_CHROME.windowRadius;
}

export function windowsTerminalInset() {
  return WINDOWS_CHROME.terminalInset;
}

// Grounded in Windows Terminal defaults (Campbell scheme, tabs-in-titlebar layout).
// References:
// - showTabsInTitlebar default true: learn.microsoft.com/windows/terminal/customize-settings/appearance
// - tab.background terminalBackground (seamless active tab): learn.microsoft.com/windows/terminal/customize-settings/themes
// - Default dark theme 1.16+: github.com/microsoft/terminal/pull/13743
// - Tab row height 36px: microsoft/terminal#9093 (MinMaxCloseControl matches tab strip)
// - Caption buttons 40px wide: TerminalApp/MinMaxCloseControl.xaml
// - Background #0C0C0C: Campbell color scheme / learn.microsoft.com terminal color schemes
// - Dark titlebar matches terminal: microsoft/terminal#14536, discussion #14844
// - Win11 window radius 8px: learn.microsoft.com windows/apps/design/basics/titlebar-design
export const WINDOWS_CHROME = {
  tabRowHeight: 36,
  windowRadius: 8,
  terminalInset: 8,
  tabRowMarginX: 8,
  tabRowMarginTop: 4,
  tabTopRadius: 5,
  tabPaddingX: 10,
  tabWidth: 168,
  appName: "tuile",
  tabIconSize: 16,
  tabIconGap: 8,
  tabCloseButtonWidth: 20,
  newTabButtonWidth: 28,
  tabMenuButtonWidth: 28,
  captionButtonWidth: 40,
  titleFontSize: 12,
  shadowBlur: 24,
  shadowOffsetY: 16,
  shadowColor: "rgba(0, 0, 0, 0.24)",
  borderDark: "rgba(255, 255, 255, 0.06)",
  borderLight: "rgba(0, 0, 0, 0.12)",
  tabTextDark: "#CCCCCC",
  tabTextLight: "#1A1A1A",
  tabRowBgDark: "#333333",
  tabRowBgLight: "#ECECEC",
  tabActiveTopAccentDark: "rgba(255, 255, 255, 0.14)",
  tabActiveTopAccentLight: "rgba(0, 0, 0, 0.12)",
  captionIconDark: "rgba(255, 255, 255, 0.9)",
  captionIconLight: "rgba(0, 0, 0, 0.9)",
  windowBgDark: "#0C0C0C",
  windowBgLight: "#F3F3F3",
};

export function windowsChromePalette(opts, renderScale = 1) {
  const frame = viewerFrameMetrics(renderScale, opts);
  const light = opts?.theme === "light";
  const windowBg = frame.termBg || (light ? WINDOWS_CHROME.windowBgLight : WINDOWS_CHROME.windowBgDark);
  const titleBarHeight = WINDOWS_CHROME.tabRowHeight * renderScale;
  return {
    windowBg,
    titleBarBg: windowBg,
    tabRowBg: light ? WINDOWS_CHROME.tabRowBgLight : WINDOWS_CHROME.tabRowBgDark,
    tabActiveBg: windowBg,
    tabActiveTopAccent: light ? WINDOWS_CHROME.tabActiveTopAccentLight : WINDOWS_CHROME.tabActiveTopAccentDark,
    border: light ? WINDOWS_CHROME.borderLight : WINDOWS_CHROME.borderDark,
    tabText: light ? WINDOWS_CHROME.tabTextLight : WINDOWS_CHROME.tabTextDark,
    captionColor: light ? WINDOWS_CHROME.captionIconLight : WINDOWS_CHROME.captionIconDark,
    titleFontSize: WINDOWS_CHROME.titleFontSize * renderScale,
    tabRowMarginX: WINDOWS_CHROME.tabRowMarginX * renderScale,
    tabRowMarginTop: WINDOWS_CHROME.tabRowMarginTop * renderScale,
    tabTopRadius: WINDOWS_CHROME.tabTopRadius * renderScale,
    tabPaddingX: WINDOWS_CHROME.tabPaddingX * renderScale,
    tabWidth: WINDOWS_CHROME.tabWidth * renderScale,
    appName: WINDOWS_CHROME.appName,
    tabIconSize: WINDOWS_CHROME.tabIconSize * renderScale,
    tabIconGap: WINDOWS_CHROME.tabIconGap * renderScale,
    tabCloseButtonWidth: WINDOWS_CHROME.tabCloseButtonWidth * renderScale,
    newTabButtonWidth: WINDOWS_CHROME.newTabButtonWidth * renderScale,
    tabMenuButtonWidth: WINDOWS_CHROME.tabMenuButtonWidth * renderScale,
    titleBarHeight,
    windowRadius: WINDOWS_CHROME.windowRadius * renderScale,
    captionButtonWidth: WINDOWS_CHROME.captionButtonWidth * renderScale,
    shadowBlur: WINDOWS_CHROME.shadowBlur * renderScale,
    shadowOffsetY: WINDOWS_CHROME.shadowOffsetY * renderScale,
    shadowColor: WINDOWS_CHROME.shadowColor,
    terminalInset: WINDOWS_CHROME.terminalInset * renderScale,
    iconScale: renderScale,
  };
}

export function osTerminalInset(osStyle) {
  if (osStyle === OS_STYLE_MACOS) {
    return macosTerminalInset();
  }
  if (osStyle === OS_STYLE_WINDOWS) {
    return windowsTerminalInset();
  }
  return 0;
}

export function defaultExportOptions(viewer = {}) {
  return {
    chrome_preset: CHROME_MINIMAL,
    background_mode: BACKGROUND_TRANSPARENT,
    background_preset: "slate",
    scale: 1,
    format: FORMAT_PNG,
    font_family: viewer.fontFamily || "'Fira Code', monospace",
    font_size_px: viewer.fontSizePx || 14,
    theme: viewer.theme || "dark",
    terminal_theme_id: viewer.terminalThemeId || "tuile:default",
    title: viewer.title || "tuile",
    show_grid_size: viewer.showGridSize ?? true,
    chrome_os_style: OS_STYLE_WIREFRAME,
  };
}

export function validateExportOptions(opts) {
  const o = normalizeChromeOptions({ ...opts });
  if (![CHROME_MINIMAL, CHROME_OS].includes(o.chrome_preset)) {
    throw new Error(`invalid chrome_preset: ${o.chrome_preset}`);
  }
  if (o.chrome_preset === CHROME_OS) {
    if (![OS_STYLE_WIREFRAME, OS_STYLE_MACOS, OS_STYLE_WINDOWS].includes(o.chrome_os_style)) {
      throw new Error(`invalid chrome_os_style: ${o.chrome_os_style}`);
    }
  }
  if (![BACKGROUND_TRANSPARENT, BACKGROUND_PRESET, BACKGROUND_CUSTOM].includes(o.background_mode)) {
    throw new Error(`invalid background_mode: ${o.background_mode}`);
  }
  if (o.background_mode === BACKGROUND_PRESET && !BACKGROUND_PRESETS[o.background_preset]) {
    throw new Error(`invalid background_preset: ${o.background_preset}`);
  }
  if (![1, 2].includes(Number(o.scale))) {
    throw new Error("scale must be 1 or 2");
  }
  if (![FORMAT_PNG, FORMAT_SVG].includes(o.format)) {
    throw new Error(`invalid format: ${o.format}`);
  }
  if (!["dark", "light"].includes(o.theme)) {
    throw new Error(`invalid theme: ${o.theme}`);
  }
  o.font_size_px = Number(o.font_size_px) || 14;
  o.show_grid_size = Boolean(o.show_grid_size);
  return o;
}

export function exportFilename(title, format) {
  const cleaned = String(title ?? "")
    .trim()
    .slice(0, 120)
    .replace(/[<>:"/\\|?*\x00-\x1f]/g, "")
    .replace(/\s+/g, " ")
    .trim();
  const base = cleaned || "tuile";
  const ext = format === FORMAT_SVG ? "svg" : "png";
  return `${base}.${ext}`;
}

export function titleBarHeight(chrome, osStyle = OS_STYLE_WIREFRAME) {
  if (chrome === CHROME_MINIMAL) {
    return 0;
  }
  if (osStyle === OS_STYLE_MACOS) {
    return macosTitleBarHeight();
  }
  if (osStyle === OS_STYLE_WINDOWS) {
    return windowsTitleBarHeight();
  }
  return 36;
}

export function chromePadding() {
  return 12;
}

export function chromeInnerGap() {
  return 8;
}

export const EXPORT_MIN_FONT_PX = 14;
export const COMPACT_SUPER_SAMPLE = 2;

export function exportScales(exportScale, viewerFontPx = 14) {
  const scale = exportScale === 2 ? 2 : 1;
  const fontPx = Math.max(Number(viewerFontPx) || 14, 8);
  const renderScale = scale === 1 ? COMPACT_SUPER_SAMPLE : scale;
  return {
    exportScale: scale,
    renderScale,
    terminalFontPx: fontPx * renderScale,
    fontPx,
    downscale: renderScale / scale,
  };
}

export function gridLabelMetrics(renderScale = 1) {
  const fontPx = GRID_LABEL.fontRem * GRID_LABEL.rootPx;
  return {
    fontSize: fontPx * renderScale,
    padX: GRID_LABEL.padX * renderScale,
    padY: GRID_LABEL.padY * renderScale,
    offsetX: GRID_LABEL.offsetX * renderScale,
    offsetY: GRID_LABEL.offsetY * renderScale,
    radius: GRID_LABEL.radius * renderScale,
  };
}

export function themeChromeAccents(presetId) {
  return THEME_CHROME_ACCENTS[presetId] || THEME_CHROME_ACCENTS.slate;
}

export function viewerFrameMetrics(renderScale = 1, opts = null) {
  const appearance = opts?.theme === "light" ? "light" : "dark";
  const termPalette = opts?.terminal_theme_id
    ? getTerminalTheme(resolveTerminalThemeId(opts.terminal_theme_id, appearance)).xterm
    : null;
  const termBg = termPalette?.background || "#0a0a0a";

  let preset = appearance === "light" ? "mist" : "slate";
  if (opts?.background_mode === BACKGROUND_PRESET && opts?.background_preset) {
    preset = opts.background_preset;
  }
  const accents = themeChromeAccents(preset);
  const frameBg = opts?.background_mode === BACKGROUND_CUSTOM ? termBg : accents.frameBg;
  return {
    framePad: VIEWER_FRAME.framePad * renderScale,
    radius: VIEWER_FRAME.radius * renderScale,
    border: accents.border,
    frameBg,
    termBg,
    labelText: accents.labelText,
    labelBg: accents.labelBg,
    labelBorder: accents.labelBorder,
  };
}

export function customBackgroundScenePad(renderScale = 1) {
  return CUSTOM_BACKGROUND_SCENE_PAD * renderScale;
}

export function expandLayoutForCustomBackground(layout, opts) {
  const renderScale = layout.renderScale ?? layout.scale ?? 1;
  const downscale = layout.downscale || 1;
  const chromeW = layout.renderOuterW;
  const chromeH = layout.renderOuterH;
  const base = {
    ...layout,
    chromeOffsetX: 0,
    chromeOffsetY: 0,
    chromeW,
    chromeH,
    scenePad: 0,
  };
  if (opts?.background_mode !== BACKGROUND_CUSTOM) {
    return base;
  }
  const scenePad = customBackgroundScenePad(renderScale);
  const renderOuterW = chromeW + scenePad * 2;
  const renderOuterH = chromeH + scenePad * 2;
  return {
    ...base,
    scenePad,
    chromeOffsetX: scenePad,
    chromeOffsetY: scenePad,
    termX: (layout.termX ?? 0) + scenePad,
    termY: (layout.termY ?? 0) + scenePad,
    renderOuterW,
    renderOuterH,
    outerW: Math.round(renderOuterW / downscale),
    outerH: Math.round(renderOuterH / downscale),
  };
}
