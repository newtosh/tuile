import {
  BACKGROUND_CUSTOM,
  BACKGROUND_PRESET,
  BACKGROUND_PRESETS,
  BACKGROUND_TRANSPARENT,
  CHROME_MINIMAL,
  CHROME_OS,
  FORMAT_SVG,
  OS_STYLE_MACOS,
  OS_STYLE_WINDOWS,
  OS_STYLE_WIREFRAME,
  chromeInnerGap,
  chromePadding,
  exportScales,
  gridLabelMetrics,
  isOsChrome,
  macosChromePalette,
  MACOS_CHROME,
  macosTitleBarHeight,
  macosTerminalInset,
  macosWindowRadius,
  windowsChromePalette,
  windowsTerminalInset,
  windowsTitleBarHeight,
  windowsWindowRadius,
  resolveOsStyle,
  titleBarHeight,
  validateExportOptions,
  viewerFrameMetrics,
} from "./export-options.js";
import { installLigatures } from "./ligatures.js";
import { getTerminalTheme, resolveTerminalThemeId } from "./terminal-themes.js";

const STROKE = "#8b8b9e";
const FRAME_FILL = "#16161a";
const TERM_BG = "#0a0a0a";
const TRAFFIC_LIGHTS = ["#ff5f57", "#febc2e", "#28c840"];

export function computeLayout(screen, opts, viewerMetrics = null) {
  const scales = exportScales(Number(opts.scale) || 1, Number(opts.font_size_px) || 14);
  const renderScale = scales.renderScale;
  const fontSize = scales.fontPx;
  const cellW = Math.max(8, Math.floor((fontSize * 6) / 10)) * renderScale;
  const cellH = (fontSize + 6) * renderScale;
  const cols = viewerMetrics?.cols || screen?.cols || Math.max(1, ...(screen?.lines || [""]).map((l) => [...l].length));
  const rows = viewerMetrics?.rows || screen?.rows || Math.max(1, (screen?.lines || []).length);
  let termW = cols * cellW;
  let termH = rows * cellH;
  if (viewerMetrics?.termW > 0 && viewerMetrics?.termH > 0) {
    termW = Math.round(viewerMetrics.termW * renderScale);
    termH = Math.round(viewerMetrics.termH * renderScale);
  }

  const base = {
    ...scales,
    scale: renderScale,
    cellW,
    cellH,
    cols,
    rows,
    termW,
    termH,
  };

  if (isOsChrome(opts)) {
    const osStyle = resolveOsStyle(opts);
    if (osStyle === OS_STYLE_MACOS || osStyle === OS_STYLE_WINDOWS) {
      const titleBar =
        (osStyle === OS_STYLE_MACOS ? macosTitleBarHeight() : windowsTitleBarHeight()) * renderScale;
      const termInset =
        (osStyle === OS_STYLE_MACOS ? macosTerminalInset() : windowsTerminalInset()) * renderScale;
      const radius =
        (osStyle === OS_STYLE_MACOS ? macosWindowRadius() : windowsWindowRadius()) * renderScale;
      const renderOuterW = termW + termInset * 2;
      const renderOuterH = titleBar + termH + termInset * 2;
      return {
        ...base,
        chrome: CHROME_OS,
        osStyle,
        titleBar,
        termInset,
        windowRadius: radius,
        termX: termInset,
        termY: titleBar + termInset,
        renderOuterW,
        renderOuterH,
        outerW: Math.round(renderOuterW / scales.downscale),
        outerH: Math.round(renderOuterH / scales.downscale),
      };
    }

    const pad = chromePadding() * renderScale;
    const title = titleBarHeight(CHROME_OS, OS_STYLE_WIREFRAME) * renderScale;
    const inner = chromeInnerGap() * renderScale;
    const renderOuterW = termW + pad * 2;
    const renderOuterH = pad + title + inner + termH + pad;
    return {
      ...base,
      chrome: CHROME_OS,
      osStyle: OS_STYLE_WIREFRAME,
      pad,
      title,
      inner,
      termX: pad,
      termY: pad + title + inner,
      renderOuterW,
      renderOuterH,
      outerW: Math.round(renderOuterW / scales.downscale),
      outerH: Math.round(renderOuterH / scales.downscale),
    };
  }

  const frame = viewerFrameMetrics(renderScale, opts);
  const frameW = termW + frame.framePad * 2;
  const frameH = termH + frame.framePad * 2;
  const renderOuterW = frameW;
  const renderOuterH = frameH;
  return {
    ...base,
    chrome: CHROME_MINIMAL,
    ...frame,
    frameW,
    frameH,
    termX: frame.framePad,
    termY: frame.framePad,
    renderOuterW,
    renderOuterH,
    outerW: Math.round(renderOuterW / scales.downscale),
    outerH: Math.round(renderOuterH / scales.downscale),
  };
}

function finalizeRenderLayout(layout, opts, termW, termH) {
  const downscale = layout.downscale || 1;
  if (isOsChrome(opts)) {
    const osStyle = resolveOsStyle(opts);
    if (osStyle === OS_STYLE_MACOS || osStyle === OS_STYLE_WINDOWS) {
      const renderScale = layout.renderScale ?? 1;
      const titleBar =
        layout.titleBar ??
        (osStyle === OS_STYLE_MACOS
          ? macosTitleBarHeight() * renderScale
          : windowsTitleBarHeight() * renderScale);
      const termInset =
        layout.termInset ??
        (osStyle === OS_STYLE_MACOS
          ? macosTerminalInset() * renderScale
          : windowsTerminalInset() * renderScale);
      const renderOuterW = termW + termInset * 2;
      const renderOuterH = titleBar + termH + termInset * 2;
      return {
        ...layout,
        termW,
        termH,
        renderOuterW,
        renderOuterH,
        outerW: Math.round(renderOuterW / downscale),
        outerH: Math.round(renderOuterH / downscale),
        termX: termInset,
        termY: titleBar + termInset,
      };
    }

    const renderOuterW = termW + layout.pad * 2;
    const renderOuterH = layout.pad + layout.title + layout.inner + termH + layout.pad;
    return {
      ...layout,
      termW,
      termH,
      renderOuterW,
      renderOuterH,
      outerW: Math.round(renderOuterW / downscale),
      outerH: Math.round(renderOuterH / downscale),
      termX: layout.pad,
      termY: layout.pad + layout.title + layout.inner,
    };
  }
  const frame = viewerFrameMetrics(layout.renderScale, opts);
  const frameW = termW + frame.framePad * 2;
  const frameH = termH + frame.framePad * 2;
  const renderOuterW = frameW;
  const renderOuterH = frameH;
  return {
    ...layout,
    ...frame,
    termW,
    termH,
    frameW,
    frameH,
    renderOuterW,
    renderOuterH,
    outerW: Math.round(renderOuterW / downscale),
    outerH: Math.round(renderOuterH / downscale),
    termX: frame.framePad,
    termY: frame.framePad,
  };
}

function downscaleCanvas(src, targetW, targetH) {
  const out = document.createElement("canvas");
  out.width = targetW;
  out.height = targetH;
  const ctx = out.getContext("2d");
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = "high";
  ctx.drawImage(src, 0, 0, targetW, targetH);
  return out;
}

function roundRectPath(ctx, x, y, w, h, r) {
  const radius = Math.min(r, w / 2, h / 2);
  ctx.beginPath();
  ctx.moveTo(x + radius, y);
  ctx.lineTo(x + w - radius, y);
  ctx.quadraticCurveTo(x + w, y, x + w, y + radius);
  ctx.lineTo(x + w, y + h - radius);
  ctx.quadraticCurveTo(x + w, y + h, x + w - radius, y + h);
  ctx.lineTo(x + radius, y + h);
  ctx.quadraticCurveTo(x, y + h, x, y + h - radius);
  ctx.lineTo(x, y + radius);
  ctx.quadraticCurveTo(x, y, x + radius, y);
  ctx.closePath();
}

function canvasSize(layout) {
  return {
    w: layout.renderOuterW ?? layout.outerW,
    h: layout.renderOuterH ?? layout.outerH,
  };
}

function drawBackground(ctx, layout, opts, bgImage) {
  const { w, h } = canvasSize(layout);
  if (opts.background_mode === BACKGROUND_TRANSPARENT) {
    ctx.clearRect(0, 0, w, h);
    return;
  }
  if (opts.background_mode === BACKGROUND_CUSTOM && bgImage) {
    ctx.drawImage(bgImage, 0, 0, w, h);
    return;
  }
  const spec = BACKGROUND_PRESETS[opts.background_preset] || BACKGROUND_PRESETS.slate;
  if (spec.kind === "solid") {
    ctx.fillStyle = spec.color;
    ctx.fillRect(0, 0, w, h);
    return;
  }
  const g = ctx.createLinearGradient(0, 0, w, h);
  g.addColorStop(0, spec.start);
  g.addColorStop(1, spec.end);
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, w, h);
}

function fillFrameCornerEars(ctx, w, h, radius, color) {
  const r = Math.min(radius, w / 2, h / 2);
  ctx.save();
  ctx.beginPath();
  ctx.rect(0, 0, w, h);
  ctx.moveTo(r, 0);
  ctx.lineTo(w - r, 0);
  ctx.quadraticCurveTo(w, 0, w, r);
  ctx.lineTo(w, h - r);
  ctx.quadraticCurveTo(w, h, w - r, h);
  ctx.lineTo(r, h);
  ctx.quadraticCurveTo(0, h, 0, h - r);
  ctx.lineTo(0, r);
  ctx.quadraticCurveTo(0, 0, r, 0);
  ctx.closePath();
  ctx.fillStyle = color;
  ctx.fill("evenodd");
  ctx.restore();
}

function drawViewerFrame(ctx, layout, opts) {
  const s = layout.renderScale ?? layout.scale;
  const w = layout.frameW;
  const h = layout.frameH;
  const transparent = opts?.background_mode === BACKGROUND_TRANSPARENT;

  if (!transparent) {
    const termBg = layout.termBg || TERM_BG;
    fillFrameCornerEars(ctx, w, h, layout.radius, termBg);
  }

  ctx.save();
  if (!transparent) {
    ctx.shadowColor = "rgba(0, 0, 0, 0.28)";
    ctx.shadowBlur = 32 * s;
    ctx.shadowOffsetY = 12 * s;
  }
  roundRectPath(ctx, 0, 0, w, h, layout.radius);
  ctx.fillStyle = layout.frameBg;
  ctx.fill();
  ctx.shadowColor = "transparent";

  roundRectPath(ctx, 0.5, 0.5, w - 1, h - 1, layout.radius);
  ctx.strokeStyle = "rgba(94, 179, 214, 0.12)";
  ctx.lineWidth = 1;
  ctx.stroke();

  roundRectPath(ctx, 0.5, 0.5, w - 1, h - 1, layout.radius);
  ctx.strokeStyle = layout.border;
  ctx.lineWidth = 1;
  ctx.stroke();
  ctx.restore();
}

function drawGridLabel(ctx, layout) {
  const label = `${layout.cols}×${layout.rows}`;
  const metrics = gridLabelMetrics(layout.renderScale ?? layout.scale);
  const { fontSize, padX, padY, offsetX, offsetY, radius } = metrics;
  ctx.font = `500 ${fontSize}px "JetBrains Mono", ui-monospace, monospace`;
  const textW = ctx.measureText(label).width;
  const boxW = textW + padX * 2;
  const boxH = fontSize + padY * 2;
  const anchorX = layout.frameW;
  const anchorY = layout.frameH;
  const x = anchorX - boxW - offsetX;
  const y = anchorY - boxH - offsetY;

  roundRectPath(ctx, x, y, boxW, boxH, radius);
  ctx.fillStyle = layout.labelBg;
  ctx.fill();
  ctx.strokeStyle = layout.labelBorder;
  ctx.lineWidth = 1;
  ctx.stroke();
  ctx.fillStyle = layout.labelText;
  ctx.textBaseline = "alphabetic";
  ctx.fillText(label, x + padX, y + padY + fontSize * 0.88);
}

function drawTrafficLights(ctx, layout, s) {
  const dot = 10 * s;
  const gap = 8 * s;
  let x = layout.pad + 10 * s;
  const cy = layout.pad + layout.title / 2;
  for (const color of TRAFFIC_LIGHTS) {
    ctx.beginPath();
    ctx.fillStyle = color;
    ctx.arc(x + dot / 2, cy, dot / 2, 0, Math.PI * 2);
    ctx.fill();
    x += dot + gap;
  }
}

function drawMacOSTrafficLights(ctx, palette) {
  const { trafficLightSize: dot, trafficLightInsetX: left, trafficLightInsetY: top, trafficLightGap: gap, trafficLights, trafficRing } = palette;
  const r = dot / 2;
  const cy = top + r;
  let x = left;
  for (const color of trafficLights) {
    const cx = x + r;
    ctx.beginPath();
    ctx.fillStyle = color;
    ctx.arc(cx, cy, r, 0, Math.PI * 2);
    ctx.fill();
    ctx.strokeStyle = trafficRing;
    ctx.lineWidth = Math.max(0.5, 0.5 * (palette.trafficLightSize / MACOS_CHROME.trafficLightSize));
    ctx.stroke();
    x += dot + gap;
  }
}

function drawWindowsActiveTab(ctx, palette, title) {
  const { titleBarHeight, tabInsetX, tabPaddingX, tabText, tabRowSeparator, titleFontSize } = palette;
  ctx.font = `400 ${titleFontSize}px "Segoe UI Variable", "Segoe UI", system-ui, sans-serif`;
  const text = title || "Terminal";
  ctx.fillStyle = tabText;
  ctx.textAlign = "left";
  ctx.textBaseline = "middle";
  ctx.fillText(text, tabInsetX + tabPaddingX, titleBarHeight / 2);
  ctx.fillStyle = tabRowSeparator;
  ctx.fillRect(0, titleBarHeight - 1, ctx.canvas.width, 1);
}

function drawWindowsCaptionButtons(ctx, palette, w) {
  const { titleBarHeight, captionButtonWidth, captionColor, iconScale } = palette;
  const btnW = captionButtonWidth;
  const kinds = ["minimize", "maximize", "close"];
  const icon = 4 * iconScale;
  ctx.strokeStyle = captionColor;
  ctx.lineWidth = Math.max(1, iconScale * 0.75);
  ctx.lineCap = "round";
  for (let i = 0; i < kinds.length; i++) {
    const x = w - (kinds.length - i) * btnW;
    const cx = x + btnW / 2;
    const cy = titleBarHeight / 2;
    if (kinds[i] === "minimize") {
      ctx.beginPath();
      ctx.moveTo(cx - icon, cy);
      ctx.lineTo(cx + icon, cy);
      ctx.stroke();
    } else if (kinds[i] === "maximize") {
      ctx.strokeRect(cx - icon, cy - icon, icon * 2, icon * 2);
    } else {
      ctx.beginPath();
      ctx.moveTo(cx - icon, cy - icon);
      ctx.lineTo(cx + icon, cy + icon);
      ctx.moveTo(cx + icon, cy - icon);
      ctx.lineTo(cx - icon, cy + icon);
      ctx.stroke();
    }
  }
}

function drawWindowsChrome(ctx, layout, opts) {
  const s = layout.renderScale ?? layout.scale ?? 1;
  const w = layout.renderOuterW;
  const h = layout.renderOuterH;
  const palette = windowsChromePalette(opts, s);
  const radius = layout.windowRadius ?? palette.windowRadius;
  const titleBar = layout.titleBar ?? palette.titleBarHeight;
  const transparent = opts?.background_mode === BACKGROUND_TRANSPARENT;

  ctx.save();
  if (!transparent) {
    ctx.shadowColor = palette.shadowColor;
    ctx.shadowBlur = palette.shadowBlur;
    ctx.shadowOffsetY = palette.shadowOffsetY;
  }
  roundRectPath(ctx, 0, 0, w, h, radius);
  ctx.fillStyle = palette.windowBg;
  ctx.fill();
  ctx.shadowColor = "transparent";
  ctx.restore();

  ctx.save();
  roundRectPath(ctx, 0, 0, w, h, radius);
  ctx.clip();
  ctx.fillStyle = palette.titleBarBg;
  ctx.fillRect(0, 0, w, h);

  drawWindowsActiveTab(ctx, palette, opts.title);
  drawWindowsCaptionButtons(ctx, palette, w);

  roundRectPath(ctx, 0.5, 0.5, w - 1, h - 1, radius);
  ctx.strokeStyle = palette.border;
  ctx.lineWidth = Math.max(0.5, 0.5 * s);
  ctx.stroke();
  ctx.restore();
}

function drawMacOSChrome(ctx, layout, opts) {
  const s = layout.renderScale ?? layout.scale ?? 1;
  const w = layout.renderOuterW;
  const h = layout.renderOuterH;
  const palette = macosChromePalette(opts, s);
  const radius = layout.windowRadius ?? palette.windowRadius;
  const titleBar = layout.titleBar ?? palette.titleBarHeight;
  const transparent = opts?.background_mode === BACKGROUND_TRANSPARENT;

  ctx.save();
  if (!transparent) {
    ctx.shadowColor = palette.shadowColor;
    ctx.shadowBlur = palette.shadowBlur;
    ctx.shadowOffsetY = palette.shadowOffsetY;
  }
  roundRectPath(ctx, 0, 0, w, h, radius);
  ctx.fillStyle = palette.windowBg;
  ctx.fill();
  ctx.shadowColor = "transparent";
  ctx.restore();

  ctx.save();
  roundRectPath(ctx, 0, 0, w, h, radius);
  ctx.clip();
  ctx.fillStyle = palette.titleBarBg;
  ctx.fillRect(0, 0, w, h);

  drawMacOSTrafficLights(ctx, palette);

  ctx.fillStyle = palette.titleColor;
  ctx.font = `500 ${palette.titleFontSize}px -apple-system, BlinkMacSystemFont, "SF Pro Text", system-ui, sans-serif`;
  ctx.textAlign = "center";
  ctx.textBaseline = "alphabetic";
  ctx.fillText(opts.title || "Terminal", w / 2, titleBar * 0.62);
  ctx.textAlign = "left";

  roundRectPath(ctx, 0.5, 0.5, w - 1, h - 1, radius);
  ctx.strokeStyle = palette.border;
  ctx.lineWidth = Math.max(0.5, 0.5 * s);
  ctx.stroke();
  ctx.restore();
}

function drawWireframeChrome(ctx, layout, opts) {
  const s = layout.scale;
  const inset = layout.pad;
  const { w, h } = canvasSize(layout);
  const transparent = opts?.background_mode === BACKGROUND_TRANSPARENT;

  if (!transparent) {
    ctx.fillStyle = FRAME_FILL;
    ctx.fillRect(0, 0, w, h);
  }
  ctx.strokeStyle = STROKE;
  ctx.lineWidth = 2 * s;
  ctx.setLineDash([5 * s, 4 * s]);
  ctx.strokeRect(inset / 2, inset / 2, w - inset, h - inset);
  ctx.beginPath();
  ctx.moveTo(inset, inset + layout.title);
  ctx.lineTo(w - inset, inset + layout.title);
  ctx.stroke();
  ctx.setLineDash([]);
  drawTrafficLights(ctx, layout, s);
  ctx.fillStyle = "#e4e4e7";
  ctx.font = `600 ${12 * s}px system-ui, sans-serif`;
  ctx.textAlign = "center";
  ctx.fillText(opts.title || "tuile", w / 2, inset + layout.title * 0.62);
  ctx.textAlign = "left";
}

function drawChrome(ctx, layout, opts) {
  if (!isOsChrome(opts)) {
    drawViewerFrame(ctx, layout, opts);
    return;
  }
  if (resolveOsStyle(opts) === OS_STYLE_MACOS) {
    drawMacOSChrome(ctx, layout, opts);
    return;
  }
  if (resolveOsStyle(opts) === OS_STYLE_WINDOWS) {
    drawWindowsChrome(ctx, layout, opts);
    return;
  }
  drawWireframeChrome(ctx, layout, opts);
}

function drawGridLabelOverlay(ctx, layout, opts) {
  if (isOsChrome(opts) || !opts.show_grid_size) {
    return;
  }
  drawGridLabel(ctx, layout);
}

function loadImageFromFile(file) {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
      URL.revokeObjectURL(url);
      resolve(img);
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error("invalid background image"));
    };
    img.src = url;
  });
}

function writeTerminal(term, data) {
  if (!data?.length) {
    return Promise.resolve();
  }
  return new Promise((resolve) => {
    term.write(data, resolve);
  });
}

function loadExportTerminal(host, layout, options, viewerMetrics) {
  const fontPx = viewerMetrics?.fontSizePx || layout.fontPx;
  const terminalFontPx = fontPx * (layout.renderScale ?? layout.scale);
  const fontFamily = viewerMetrics?.fontFamily || options.font_family;
  const appearance = options.theme === "light" ? "light" : "dark";
  const themeId = resolveTerminalThemeId(options.terminal_theme_id, appearance);
  const palette = getTerminalTheme(themeId).xterm;
  const term = new Terminal({
    cols: viewerMetrics?.cols || layout.cols,
    rows: viewerMetrics?.rows || layout.rows,
    fontFamily,
    fontSize: terminalFontPx,
    lineHeight: 1,
    letterSpacing: 0,
    customGlyphs: true,
    drawBoldTextInBrightColors: true,
    scrollback: 0,
    convertEol: true,
    allowProposedApi: true,
    theme: palette,
  });
  term.open(host);

  let removeLigatures = null;
  if (window.Unicode11Addon) {
    const unicode11Addon = new Unicode11Addon.Unicode11Addon();
    term.loadAddon(unicode11Addon);
    term.unicode.activeVersion = "11";
  }
  if (window.CanvasAddon) {
    const canvasAddon = new CanvasAddon.CanvasAddon();
    term.loadAddon(canvasAddon);
  }
  removeLigatures = installLigatures(term);

  return { term, removeLigatures };
}

function compositeTerminalCanvases(host) {
  const canvases = [...host.querySelectorAll(".xterm-screen canvas")];
  if (!canvases.length) {
    return host.querySelector("canvas");
  }
  if (canvases.length === 1) {
    return canvases[0];
  }
  const out = document.createElement("canvas");
  out.width = canvases[0].width;
  out.height = canvases[0].height;
  const ctx = out.getContext("2d");
  for (const canvas of canvases) {
    ctx.drawImage(canvas, 0, 0);
  }
  return out;
}

function measureExportTerminal(term, termCanvas) {
  const css = term?._core?._renderService?.dimensions?.css;
  if (css?.canvas?.width > 0 && css?.canvas?.height > 0) {
    return {
      width: Math.round(css.canvas.width),
      height: Math.round(css.canvas.height),
      canvas: termCanvas,
    };
  }
  const dpr = window.devicePixelRatio || 1;
  return {
    width: Math.round((termCanvas?.width || 0) / dpr),
    height: Math.round((termCanvas?.height || 0) / dpr),
    canvas: termCanvas,
  };
}

function drawExportTerminal(ctx, measured, layout) {
  const { width, height, canvas } = measured;
  if (!canvas) {
    return;
  }
  const termBg = layout.termBg || TERM_BG;
  ctx.fillStyle = termBg;
  ctx.fillRect(layout.termX, layout.termY, width, height);
  ctx.drawImage(canvas, 0, 0, canvas.width, canvas.height, layout.termX, layout.termY, width, height);
}

function drawTerminalFallback(ctx, screen, layout) {
  const fontSize = layout.cellH - 6;
  ctx.font = `${fontSize}px monospace`;
  ctx.fillStyle = "#e4e4e4";
  const lines = screen?.lines || [];
  lines.forEach((line, y) => {
    ctx.fillText(line, layout.termX + 4, layout.termY + (y + 1) * layout.cellH - 4);
  });
}

async function waitForTerminalCanvas(term, host) {
  let termCanvas = null;
  let measured = { width: 0, height: 0, canvas: null };
  for (let attempt = 0; attempt < 10; attempt++) {
    term.refresh(0, term.rows - 1);
    await new Promise((resolve) => requestAnimationFrame(resolve));
    termCanvas = compositeTerminalCanvases(host);
    measured = measureExportTerminal(term, termCanvas);
    if (measured.width > 0 && measured.height > 0 && termCanvas) {
      return { termCanvas, measured };
    }
  }
  return { termCanvas, measured };
}

export async function composeExportPNG({ screen, replayBytes, opts, backgroundFile, viewerMetrics }) {
  const options = validateExportOptions(opts);
  const layout = computeLayout(screen, options, viewerMetrics);
  let bgImage = null;
  if (options.background_mode === BACKGROUND_CUSTOM && backgroundFile) {
    bgImage = await loadImageFromFile(backgroundFile);
  }

  const host = document.createElement("div");
  host.style.cssText = "position:fixed;left:0;top:0;opacity:0;pointer-events:none;z-index:-1;";
  document.body.appendChild(host);

  const { term, removeLigatures } = loadExportTerminal(host, layout, options, viewerMetrics);
  try {
    if (replayBytes?.length) {
      await writeTerminal(term, replayBytes);
    } else if (screen?.lines) {
      await writeTerminal(term, screen.lines.join("\n"));
    }
    term.refresh(0, term.rows - 1);
    if (document.fonts?.ready) {
      await document.fonts.ready;
    }

    const { termCanvas, measured } = await waitForTerminalCanvas(term, host);
    let termW = layout.termW;
    let termH = layout.termH;
    if (viewerMetrics?.termW > 0 && viewerMetrics?.termH > 0) {
      termW = Math.round(viewerMetrics.termW * layout.renderScale);
      termH = Math.round(viewerMetrics.termH * layout.renderScale);
    } else if (measured.width > 0 && measured.height > 0) {
      termW = measured.width;
      termH = measured.height;
    }
    const renderLayout = finalizeRenderLayout(layout, options, termW, termH);

    const out = document.createElement("canvas");
    out.width = renderLayout.renderOuterW;
    out.height = renderLayout.renderOuterH;
    const ctx = out.getContext("2d");
    drawBackground(ctx, renderLayout, options, bgImage);
    drawChrome(ctx, renderLayout, options);
    if (termCanvas) {
      drawExportTerminal(ctx, { ...measured, width: termW, height: termH }, renderLayout);
    } else {
      drawTerminalFallback(ctx, screen, renderLayout);
    }
    drawGridLabelOverlay(ctx, renderLayout, options);

    const exportCanvas =
      renderLayout.outerW === out.width && renderLayout.outerH === out.height
        ? out
        : downscaleCanvas(out, renderLayout.outerW, renderLayout.outerH);

    const blob = await new Promise((resolve, reject) => {
      exportCanvas.toBlob((b) => (b ? resolve(b) : reject(new Error("png encode failed"))), "image/png");
    });
    return blob;
  } finally {
    removeLigatures?.();
    term.dispose();
    host.remove();
  }
}

export async function composeExport({ screen, replayBytes, opts, backgroundFile, viewerMetrics }) {
  const options = validateExportOptions(opts);
  if (options.format === FORMAT_SVG) {
    return composeExportSVG({ screen, opts: options, viewerMetrics });
  }
  return composeExportPNG({ screen, replayBytes, opts: options, backgroundFile, viewerMetrics });
}

export async function composeExportSVG({ screen, opts, viewerMetrics }) {
  const options = validateExportOptions({ ...opts, format: FORMAT_SVG });
  const layout = computeLayout(screen, options, viewerMetrics);
  const lines = screen?.lines || [];
  const osChrome = isOsChrome(options);
  const osStyle = resolveOsStyle(options);
  let svg = `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" width="${layout.outerW}" height="${layout.outerH}" viewBox="0 0 ${layout.renderOuterW} ${layout.renderOuterH}">`;

  if (options.background_mode === BACKGROUND_PRESET) {
    const spec = BACKGROUND_PRESETS[options.background_preset] || BACKGROUND_PRESETS.slate;
    if (spec.kind === "solid") {
      svg += `<rect width="100%" height="100%" fill="${spec.color}"/>`;
    } else {
      svg += `<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="${spec.start}"/><stop offset="100%" stop-color="${spec.end}"/></linearGradient></defs><rect width="100%" height="100%" fill="url(#g)"/>`;
    }
  }

  if (osChrome && osStyle === OS_STYLE_MACOS) {
    const palette = macosChromePalette(options, layout.renderScale);
    const radius = layout.windowRadius;
    const titleBar = layout.titleBar;
    const dot = palette.trafficLightSize;
    const gap = palette.trafficLightGap;
    let x = palette.trafficLightInsetX;
    const cy = palette.trafficLightInsetY + dot / 2;
    svg += `<rect x="0" y="0" width="${layout.renderOuterW}" height="${layout.renderOuterH}" rx="${radius}" fill="${palette.windowBg}" stroke="${palette.border}" stroke-width="0.5"/>`;
    for (const color of palette.trafficLights) {
      svg += `<circle cx="${x + dot / 2}" cy="${cy}" r="${dot / 2}" fill="${color}" stroke="${palette.trafficRing}" stroke-width="0.5"/>`;
      x += dot + gap;
    }
    svg += `<text x="${layout.renderOuterW / 2}" y="${titleBar * 0.62}" text-anchor="middle" fill="${palette.titleColor}" font-family="-apple-system,BlinkMacSystemFont,&quot;SF Pro Text&quot;,system-ui,sans-serif" font-size="${palette.titleFontSize}" font-weight="500">${escapeXml(options.title || "Terminal")}</text>`;
  } else if (osChrome && osStyle === OS_STYLE_WINDOWS) {
    const palette = windowsChromePalette(options, layout.renderScale);
    const radius = layout.windowRadius;
    const titleBar = layout.titleBar;
    const btnW = palette.captionButtonWidth;
    const icon = 4 * palette.iconScale;
    const text = options.title || "Terminal";
    svg += `<rect x="0" y="0" width="${layout.renderOuterW}" height="${layout.renderOuterH}" rx="${radius}" fill="${palette.windowBg}" stroke="${palette.border}" stroke-width="0.5"/>`;
    svg += `<text x="${palette.tabInsetX + palette.tabPaddingX}" y="${titleBar / 2}" dominant-baseline="middle" fill="${palette.tabText}" font-family=&quot;Segoe UI Variable&quot;,&quot;Segoe UI&quot;,system-ui,sans-serif font-size="${palette.titleFontSize}" font-weight="400">${escapeXml(text)}</text>`;
    svg += `<rect x="0" y="${titleBar - 1}" width="${layout.renderOuterW}" height="1" fill="${palette.tabRowSeparator}"/>`;
    const kinds = ["minimize", "maximize", "close"];
    for (let i = 0; i < kinds.length; i++) {
      const x = layout.renderOuterW - (kinds.length - i) * btnW;
      const cx = x + btnW / 2;
      const cy = titleBar / 2;
      if (kinds[i] === "minimize") {
        svg += `<line x1="${cx - icon}" y1="${cy}" x2="${cx + icon}" y2="${cy}" stroke="${palette.captionColor}" stroke-width="${Math.max(1, palette.iconScale * 0.75)}"/>`;
      } else if (kinds[i] === "maximize") {
        svg += `<rect x="${cx - icon}" y="${cy - icon}" width="${icon * 2}" height="${icon * 2}" fill="none" stroke="${palette.captionColor}" stroke-width="${Math.max(1, palette.iconScale * 0.75)}"/>`;
      } else {
        svg += `<line x1="${cx - icon}" y1="${cy - icon}" x2="${cx + icon}" y2="${cy + icon}" stroke="${palette.captionColor}" stroke-width="${Math.max(1, palette.iconScale * 0.75)}"/>`;
        svg += `<line x1="${cx + icon}" y1="${cy - icon}" x2="${cx - icon}" y2="${cy + icon}" stroke="${palette.captionColor}" stroke-width="${Math.max(1, palette.iconScale * 0.75)}"/>`;
      }
    }
  } else if (osChrome) {
    const s = layout.renderScale;
    const stroke = STROKE;
    const dash = `${5 * s} ${4 * s}`;
    if (options.background_mode !== BACKGROUND_TRANSPARENT) {
      svg += `<rect width="${layout.renderOuterW}" height="${layout.renderOuterH}" fill="${FRAME_FILL}" stroke="${stroke}" stroke-width="${2 * s}" stroke-dasharray="${dash}"/>`;
    } else {
      svg += `<rect width="${layout.renderOuterW}" height="${layout.renderOuterH}" fill="none" stroke="${stroke}" stroke-width="${2 * s}" stroke-dasharray="${dash}"/>`;
    }
    const y = layout.pad + layout.title;
    svg += `<line x1="${layout.pad}" y1="${y}" x2="${layout.renderOuterW - layout.pad}" y2="${y}" stroke="${stroke}" stroke-width="${2 * s}" stroke-dasharray="${dash}"/>`;
    const dot = 10 * s;
    const gap = 8 * s;
    let x = layout.pad + 10 * s;
    const cy = layout.pad + layout.title / 2;
    for (const color of TRAFFIC_LIGHTS) {
      svg += `<circle cx="${x + dot / 2}" cy="${cy}" r="${dot / 2}" fill="${color}"/>`;
      x += dot + gap;
    }
    svg += `<text x="${layout.renderOuterW / 2}" y="${layout.pad + layout.title * 0.62}" text-anchor="middle" fill="#e4e4e7" font-family="system-ui" font-size="${12 * s}" font-weight="600">${escapeXml(options.title || "tuile")}</text>`;
  } else {
    if (options.background_mode !== BACKGROUND_TRANSPARENT) {
      svg += `<rect x="0" y="0" width="${layout.frameW}" height="${layout.frameH}" fill="${layout.termBg || TERM_BG}"/>`;
    }
    svg += `<rect x="0" y="0" width="${layout.frameW}" height="${layout.frameH}" rx="${layout.radius}" fill="${layout.frameBg}" stroke="${layout.border}" stroke-width="1"/>`;
  }

  svg += `<g transform="translate(${layout.termX},${layout.termY})"><rect width="${layout.termW}" height="${layout.termH}" fill="${TERM_BG}"/>`;
  const termFontSize = layout.fontPx * layout.renderScale;
  lines.forEach((line, y) => {
    svg += `<text x="4" y="${(y + 1) * layout.cellH - 4}" fill="#e4e4e4" font-family="monospace" font-size="${termFontSize}">${escapeXml(line)}</text>`;
  });
  svg += `</g>`;

  if (!osChrome && options.show_grid_size) {
    const label = `${layout.cols}×${layout.rows}`;
    const badge = gridLabelMetrics(layout.renderScale);
    const anchorX = layout.frameW;
    const anchorY = layout.frameH;
    const boxW = label.length * badge.fontSize * 0.62 + badge.padX * 2;
    const boxH = badge.fontSize + badge.padY * 2;
    const lx = anchorX - boxW - badge.offsetX;
    const ly = anchorY - boxH - badge.offsetY;
    svg += `<rect x="${lx}" y="${ly}" width="${boxW}" height="${boxH}" rx="${badge.radius}" fill="${layout.labelBg}" stroke="${layout.labelBorder}" stroke-width="1"/>`;
    svg += `<text x="${lx + badge.padX}" y="${ly + badge.padY + badge.fontSize * 0.88}" fill="${layout.labelText}" font-family="JetBrains Mono, ui-monospace, monospace" font-size="${badge.fontSize}" font-weight="500">${escapeXml(label)}</text>`;
  }

  svg += `</svg>`;
  return new Blob([svg], { type: "image/svg+xml" });
}

function escapeXml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

export function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
