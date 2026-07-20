export const CHROME_MINIMAL = "minimal";
export const CHROME_OS_WIREFRAME = "os-wireframe";

export const BACKGROUND_TRANSPARENT = "transparent";
export const BACKGROUND_PRESET = "preset";
export const BACKGROUND_CUSTOM = "custom";

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

// Matches web/style.css observe-mode terminal frame (GRID_FRAME_PAD in app.js).
export const VIEWER_FRAME = {
  border: "#334e60",
  frameBg: "#0b0c0f",
  labelText: "#8ec8e0",
  labelBg: "rgba(22, 22, 28, 0.88)",
  labelBorder: "rgba(51, 78, 96, 0.55)",
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

export function defaultExportOptions(viewer = {}) {
  return {
    chrome_preset: CHROME_MINIMAL,
    background_mode: BACKGROUND_PRESET,
    background_preset: "slate",
    scale: 1,
    format: FORMAT_PNG,
    font_family: viewer.fontFamily || "'Fira Code', monospace",
    font_size_px: viewer.fontSizePx || 14,
    theme: "dark",
    title: viewer.title || "tuile",
    show_grid_size: viewer.showGridSize ?? true,
  };
}

export function validateExportOptions(opts) {
  const o = { ...opts };
  if (![CHROME_MINIMAL, CHROME_OS_WIREFRAME].includes(o.chrome_preset)) {
    throw new Error(`invalid chrome_preset: ${o.chrome_preset}`);
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

export function titleBarHeight(chrome) {
  return chrome === CHROME_OS_WIREFRAME ? 36 : 0;
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

export function viewerFrameMetrics(renderScale = 1) {
  return {
    framePad: VIEWER_FRAME.framePad * renderScale,
    radius: VIEWER_FRAME.radius * renderScale,
    border: VIEWER_FRAME.border,
    frameBg: VIEWER_FRAME.frameBg,
  };
}
