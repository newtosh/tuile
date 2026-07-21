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
  expandLayoutForCustomBackground,
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
      return expandLayoutForCustomBackground(
        {
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
        },
        opts
      );
    }

    const pad = chromePadding() * renderScale;
    const title = titleBarHeight(CHROME_OS, OS_STYLE_WIREFRAME) * renderScale;
    const inner = chromeInnerGap() * renderScale;
    const renderOuterW = termW + pad * 2;
    const renderOuterH = pad + title + inner + termH + pad;
    return expandLayoutForCustomBackground(
      {
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
      },
      opts
    );
  }

  const frame = viewerFrameMetrics(renderScale, opts);
  const frameW = termW + frame.framePad * 2;
  const frameH = termH + frame.framePad * 2;
  const renderOuterW = frameW;
  const renderOuterH = frameH;
  return expandLayoutForCustomBackground(
    {
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
    },
    opts
  );
}

function chromeRect(layout) {
  return {
    x: layout.chromeOffsetX ?? 0,
    y: layout.chromeOffsetY ?? 0,
    w: layout.chromeW ?? layout.renderOuterW,
    h: layout.chromeH ?? layout.renderOuterH,
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
      return expandLayoutForCustomBackground(
        {
          ...layout,
          termW,
          termH,
          renderOuterW,
          renderOuterH,
          outerW: Math.round(renderOuterW / downscale),
          outerH: Math.round(renderOuterH / downscale),
          termX: termInset,
          termY: titleBar + termInset,
        },
        opts
      );
    }

    const renderOuterW = termW + layout.pad * 2;
    const renderOuterH = layout.pad + layout.title + layout.inner + termH + layout.pad;
    return expandLayoutForCustomBackground(
      {
        ...layout,
        termW,
        termH,
        renderOuterW,
        renderOuterH,
        outerW: Math.round(renderOuterW / downscale),
        outerH: Math.round(renderOuterH / downscale),
        termX: layout.pad,
        termY: layout.pad + layout.title + layout.inner,
      },
      opts
    );
  }
  const frame = viewerFrameMetrics(layout.renderScale, opts);
  const frameW = termW + frame.framePad * 2;
  const frameH = termH + frame.framePad * 2;
  const renderOuterW = frameW;
  const renderOuterH = frameH;
  return expandLayoutForCustomBackground(
    {
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
    },
    opts
  );
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

function roundRectTopPath(ctx, x, y, w, h, r) {
  const radius = Math.min(r, w / 2, h);
  ctx.beginPath();
  ctx.moveTo(x, y + h);
  ctx.lineTo(x, y + radius);
  ctx.quadraticCurveTo(x, y, x + radius, y);
  ctx.lineTo(x + w - radius, y);
  ctx.quadraticCurveTo(x + w, y, x + w, y + radius);
  ctx.lineTo(x + w, y + h);
  ctx.closePath();
}

function fillRoundRectTop(ctx, x, y, w, h, r, color) {
  ctx.fillStyle = color;
  roundRectTopPath(ctx, x, y, w, h, r);
  ctx.fill();
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
  if (opts.background_mode === BACKGROUND_CUSTOM) {
    if (bgImage) {
      ctx.drawImage(bgImage, 0, 0, w, h);
    }
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
  const { x, y, w, h } = chromeRect(layout);
  const frameW = layout.frameW && !layout.scenePad ? layout.frameW : w;
  const frameH = layout.frameH && !layout.scenePad ? layout.frameH : h;
  const solidFrameFill = opts?.background_mode !== BACKGROUND_CUSTOM;

  if (solidFrameFill) {
    const termBg = layout.termBg || TERM_BG;
    fillFrameCornerEars(ctx, x, y, frameW, frameH, layout.radius, termBg);
  }

  ctx.save();
  if (solidFrameFill) {
    ctx.shadowColor = "rgba(0, 0, 0, 0.28)";
    ctx.shadowBlur = 32 * s;
    ctx.shadowOffsetY = 12 * s;
  }
  roundRectPath(ctx, x, y, frameW, frameH, layout.radius);
  if (solidFrameFill) {
    ctx.fillStyle = layout.frameBg;
    ctx.fill();
  }
  ctx.shadowColor = "transparent";

  roundRectPath(ctx, x + 0.5, y + 0.5, frameW - 1, frameH - 1, layout.radius);
  ctx.strokeStyle = "rgba(94, 179, 214, 0.12)";
  ctx.lineWidth = 1;
  ctx.stroke();

  roundRectPath(ctx, x + 0.5, y + 0.5, frameW - 1, frameH - 1, layout.radius);
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
  const chrome = chromeRect(layout);
  const frameW = layout.frameW && !layout.scenePad ? layout.frameW : chrome.w;
  const frameH = layout.frameH && !layout.scenePad ? layout.frameH : chrome.h;
  const anchorX = chrome.x + frameW;
  const anchorY = chrome.y + frameH;
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

function drawTrafficLights(ctx, layout, s, ox = 0, oy = 0) {
  const dot = 10 * s;
  const gap = 8 * s;
  let x = ox + layout.pad + 10 * s;
  const cy = oy + layout.pad + layout.title / 2;
  for (const color of TRAFFIC_LIGHTS) {
    ctx.beginPath();
    ctx.fillStyle = color;
    ctx.arc(x + dot / 2, cy, dot / 2, 0, Math.PI * 2);
    ctx.fill();
    x += dot + gap;
  }
}

function drawMacOSTrafficLights(ctx, palette, ox = 0, oy = 0) {
  const { trafficLightSize: dot, trafficLightInsetX: left, trafficLightInsetY: top, trafficLightGap: gap, trafficLights, trafficRing } = palette;
  const r = dot / 2;
  const cy = oy + top + r;
  let x = ox + left;
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

function drawTuileFavicon(ctx, x, y, size, image) {
  if (image) {
    ctx.drawImage(image, x, y, size, size);
    return;
  }
  const u = size / 32;
  roundRectPath(ctx, x, y, size, size, 6 * u);
  ctx.fillStyle = "#0c0c0e";
  ctx.fill();
  const squares = [
    { x: 6, y: 6, o: 1 },
    { x: 17, y: 6, o: 0.82 },
    { x: 6, y: 17, o: 0.82 },
    { x: 17, y: 17, o: 1 },
  ];
  for (const sq of squares) {
    ctx.save();
    ctx.globalAlpha = sq.o;
    ctx.fillStyle = "#e8a54b";
    roundRectPath(ctx, x + sq.x * u, y + sq.y * u, 9 * u, 9 * u, 1.5 * u);
    ctx.fill();
    ctx.restore();
  }
}

function drawWindowsTabCloseButton(ctx, palette, tabX, tabY, tabW, tabH) {
  const { tabPaddingX, tabCloseButtonWidth, captionColor, iconScale } = palette;
  const cx = tabX + tabW - tabPaddingX - tabCloseButtonWidth / 2;
  const cy = tabY + tabH / 2;
  const icon = 3.5 * iconScale;
  ctx.strokeStyle = captionColor;
  ctx.lineWidth = Math.max(1, iconScale * 0.75);
  ctx.lineCap = "round";
  ctx.beginPath();
  ctx.moveTo(cx - icon, cy - icon);
  ctx.lineTo(cx + icon, cy + icon);
  ctx.moveTo(cx + icon, cy - icon);
  ctx.lineTo(cx - icon, cy + icon);
  ctx.stroke();
}

function drawWindowsNewTabButton(ctx, palette, x, cy) {
  const { newTabButtonWidth, captionColor, iconScale } = palette;
  const cx = x + newTabButtonWidth / 2;
  const icon = 5 * iconScale;
  ctx.strokeStyle = captionColor;
  ctx.lineWidth = Math.max(1, iconScale * 0.75);
  ctx.lineCap = "round";
  ctx.beginPath();
  ctx.moveTo(cx - icon, cy);
  ctx.lineTo(cx + icon, cy);
  ctx.moveTo(cx, cy - icon);
  ctx.lineTo(cx, cy + icon);
  ctx.stroke();
}

function drawWindowsTabMenuChevron(ctx, palette, x, cy) {
  const { tabMenuButtonWidth, captionColor, iconScale } = palette;
  const cx = x + tabMenuButtonWidth / 2;
  const half = 3.5 * iconScale;
  ctx.strokeStyle = captionColor;
  ctx.lineWidth = Math.max(1, iconScale * 0.75);
  ctx.lineCap = "round";
  ctx.lineJoin = "round";
  ctx.beginPath();
  ctx.moveTo(cx - half, cy - half * 0.35);
  ctx.lineTo(cx, cy + half * 0.65);
  ctx.lineTo(cx + half, cy - half * 0.35);
  ctx.stroke();
}

function drawWindowsTabRow(ctx, palette, faviconImage, ox = 0, oy = 0) {
  const {
    titleBarHeight,
    tabRowMarginX,
    tabRowMarginTop,
    tabTopRadius,
    tabPaddingX,
    tabWidth,
    tabIconSize,
    tabIconGap,
    tabText,
    tabActiveBg,
    tabActiveTopAccent,
    titleFontSize,
    appName,
    newTabButtonWidth,
    tabCloseButtonWidth,
  } = palette;
  const tabX = ox + tabRowMarginX;
  const tabY = oy + tabRowMarginTop;
  const tabW = tabWidth;
  const tabH = titleBarHeight - tabRowMarginTop;
  fillRoundRectTop(ctx, tabX, tabY, tabW, tabH, tabTopRadius, tabActiveBg);
  ctx.save();
  roundRectTopPath(ctx, tabX, tabY, tabW, tabH, tabTopRadius);
  ctx.clip();
  ctx.fillStyle = tabActiveTopAccent;
  ctx.fillRect(tabX, tabY, tabW, Math.max(1, palette.iconScale ?? 1));
  ctx.restore();
  const iconY = tabY + (tabH - tabIconSize) / 2;
  const iconX = tabX + tabPaddingX;
  drawTuileFavicon(ctx, iconX, iconY, tabIconSize, faviconImage);
  const textX = iconX + tabIconSize + tabIconGap;
  const textMaxW = tabW - tabPaddingX * 2 - tabIconSize - tabIconGap - tabCloseButtonWidth;
  ctx.font = `400 ${titleFontSize}px "Segoe UI Variable", "Segoe UI", system-ui, sans-serif`;
  ctx.fillStyle = tabText;
  ctx.textAlign = "left";
  ctx.textBaseline = "middle";
  ctx.save();
  ctx.beginPath();
  ctx.rect(textX, tabY, Math.max(0, textMaxW), tabH);
  ctx.clip();
  ctx.fillText(appName, textX, tabY + tabH / 2);
  ctx.restore();
  drawWindowsTabCloseButton(ctx, palette, tabX, tabY, tabW, tabH);
  const controlsCy = tabY + tabH / 2;
  const controlsX = tabX + tabW;
  drawWindowsNewTabButton(ctx, palette, controlsX, controlsCy);
  drawWindowsTabMenuChevron(ctx, palette, controlsX + newTabButtonWidth, controlsCy);
}

function drawWindowsCaptionButtons(ctx, palette, w, ox = 0, oy = 0) {
  const { titleBarHeight, captionButtonWidth, captionColor, iconScale } = palette;
  const btnW = captionButtonWidth;
  const kinds = ["minimize", "maximize", "close"];
  const icon = 4 * iconScale;
  ctx.strokeStyle = captionColor;
  ctx.lineWidth = Math.max(1, iconScale * 0.75);
  ctx.lineCap = "round";
  for (let i = 0; i < kinds.length; i++) {
    const x = ox + w - (kinds.length - i) * btnW;
    const cx = x + btnW / 2;
    const cy = oy + titleBarHeight / 2;
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

function drawWindowsChrome(ctx, layout, opts, faviconImage = null) {
  const s = layout.renderScale ?? layout.scale ?? 1;
  const { x, y, w, h } = chromeRect(layout);
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
  roundRectPath(ctx, x, y, w, h, radius);
  ctx.fillStyle = palette.windowBg;
  ctx.fill();
  ctx.shadowColor = "transparent";
  ctx.restore();

  ctx.save();
  roundRectPath(ctx, x, y, w, h, radius);
  ctx.clip();
  ctx.fillStyle = palette.tabRowBg;
  ctx.fillRect(x, y, w, titleBar);
  ctx.fillStyle = palette.windowBg;
  ctx.fillRect(x, y + titleBar, w, h - titleBar);

  drawWindowsTabRow(ctx, palette, faviconImage, x, y);
  drawWindowsCaptionButtons(ctx, palette, w, x, y);

  roundRectPath(ctx, x + 0.5, y + 0.5, w - 1, h - 1, radius);
  ctx.strokeStyle = palette.border;
  ctx.lineWidth = Math.max(0.5, 0.5 * s);
  ctx.stroke();
  ctx.restore();
}

function drawMacOSChrome(ctx, layout, opts) {
  const s = layout.renderScale ?? layout.scale ?? 1;
  const { x, y, w, h } = chromeRect(layout);
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
  roundRectPath(ctx, x, y, w, h, radius);
  ctx.fillStyle = palette.windowBg;
  ctx.fill();
  ctx.shadowColor = "transparent";
  ctx.restore();

  ctx.save();
  roundRectPath(ctx, x, y, w, h, radius);
  ctx.clip();
  ctx.fillStyle = palette.titleBarBg;
  ctx.fillRect(x, y, w, h);

  drawMacOSTrafficLights(ctx, palette, x, y);

  ctx.fillStyle = palette.titleColor;
  ctx.font = `500 ${palette.titleFontSize}px -apple-system, BlinkMacSystemFont, "SF Pro Text", system-ui, sans-serif`;
  ctx.textAlign = "center";
  ctx.textBaseline = "alphabetic";
  ctx.fillText(opts.title || "Terminal", x + w / 2, y + titleBar * 0.62);
  ctx.textAlign = "left";

  roundRectPath(ctx, x + 0.5, y + 0.5, w - 1, h - 1, radius);
  ctx.strokeStyle = palette.border;
  ctx.lineWidth = Math.max(0.5, 0.5 * s);
  ctx.stroke();
  ctx.restore();
}

function drawWireframeChrome(ctx, layout, opts) {
  const s = layout.scale;
  const inset = layout.pad;
  const { x, y, w, h } = chromeRect(layout);
  const solidFrameFill = opts?.background_mode !== BACKGROUND_CUSTOM;

  if (solidFrameFill) {
    ctx.fillStyle = FRAME_FILL;
    ctx.fillRect(x, y, w, h);
  }
  ctx.strokeStyle = STROKE;
  ctx.lineWidth = 2 * s;
  ctx.setLineDash([5 * s, 4 * s]);
  ctx.strokeRect(x + inset / 2, y + inset / 2, w - inset, h - inset);
  ctx.beginPath();
  ctx.moveTo(x + inset, y + inset + layout.title);
  ctx.lineTo(x + w - inset, y + inset + layout.title);
  ctx.stroke();
  ctx.setLineDash([]);
  drawTrafficLights(ctx, layout, s, x, y);
  ctx.fillStyle = "#e4e4e7";
  ctx.font = `600 ${12 * s}px system-ui, sans-serif`;
  ctx.textAlign = "center";
  ctx.fillText(opts.title || "tuile", x + w / 2, y + inset + layout.title * 0.62);
  ctx.textAlign = "left";
}

function drawChrome(ctx, layout, opts, assets = {}) {
  if (!isOsChrome(opts)) {
    drawViewerFrame(ctx, layout, opts);
    return;
  }
  if (resolveOsStyle(opts) === OS_STYLE_MACOS) {
    drawMacOSChrome(ctx, layout, opts);
    return;
  }
  if (resolveOsStyle(opts) === OS_STYLE_WINDOWS) {
    drawWindowsChrome(ctx, layout, opts, assets.favicon);
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

function loadImageFromURL(src) {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error(`failed to load image: ${src}`));
    img.src = src;
  });
}

let faviconImagePromise;
async function loadFaviconImage() {
  if (!faviconImagePromise) {
    faviconImagePromise = (async () => {
      for (const src of ["/assets/favicon.png", "/assets/favicon.svg"]) {
        try {
          return await loadImageFromURL(src);
        } catch {
          // try next asset
        }
      }
      return null;
    })();
  }
  return faviconImagePromise;
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

function flattenCanvasForSvgEmbed(src, fill = "#0b0c0f") {
  const canvas = document.createElement("canvas");
  canvas.width = src.width;
  canvas.height = src.height;
  const ctx = canvas.getContext("2d", { alpha: false });
  if (!ctx) {
    throw new Error("opaque 2d context unavailable");
  }
  ctx.fillStyle = fill;
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.drawImage(src, 0, 0);
  return canvas;
}

function wrapRasterSvg(dataUrl, logicalW, logicalH) {
  return `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" width="${logicalW}" height="${logicalH}" viewBox="0 0 ${logicalW} ${logicalH}"><image x="0" y="0" width="${logicalW}" height="${logicalH}" href="${dataUrl}"/></svg>`;
}

function svgEmbedBackgroundFill(opts) {
  if (opts.background_mode === BACKGROUND_TRANSPARENT) {
    return null;
  }
  if (opts.background_mode === BACKGROUND_CUSTOM) {
    return null;
  }
  if (opts.background_mode === BACKGROUND_PRESET) {
    const spec = BACKGROUND_PRESETS[opts.background_preset] || BACKGROUND_PRESETS.slate;
    if (spec.kind === "solid") {
      return spec.color;
    }
  }
  return "#0b0c0f";
}

async function canvasToDataUrl(canvas, mimeType = "image/png", quality) {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          reject(new Error("canvas encode failed"));
          return;
        }
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result));
        reader.onerror = () => reject(reader.error || new Error("read failed"));
        reader.readAsDataURL(blob);
      },
      mimeType,
      quality
    );
  });
}

async function prepareExportTerminal({ screen, replayBytes, opts, viewerMetrics }) {
  const options = validateExportOptions(opts);
  const layout = computeLayout(screen, options, viewerMetrics);
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
    return {
      options,
      renderLayout,
      termCanvas,
      termW,
      termH,
      measured,
      dispose() {
        removeLigatures?.();
        term.dispose();
        host.remove();
      },
    };
  } catch (err) {
    removeLigatures?.();
    term.dispose();
    host.remove();
    throw err;
  }
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

async function composeExportRasterCanvas({
  screen,
  replayBytes,
  opts,
  backgroundFile,
  viewerMetrics,
  skipDownscale = false,
}) {
  let bgImage = null;
  const options = validateExportOptions(opts);
  if (options.background_mode === BACKGROUND_CUSTOM && backgroundFile) {
    bgImage = await loadImageFromFile(backgroundFile);
  }

  const prepared = await prepareExportTerminal({ screen, replayBytes, opts, viewerMetrics });
  try {
    const { renderLayout, termCanvas, termW, termH, measured } = prepared;
    const out = document.createElement("canvas");
    out.width = renderLayout.renderOuterW;
    out.height = renderLayout.renderOuterH;
    const ctx = out.getContext("2d");
    drawBackground(ctx, renderLayout, prepared.options, bgImage);
    const chromeAssets =
      resolveOsStyle(prepared.options) === OS_STYLE_WINDOWS ? { favicon: await loadFaviconImage() } : {};
    drawChrome(ctx, renderLayout, prepared.options, chromeAssets);
    if (termCanvas) {
      drawExportTerminal(ctx, { ...measured, width: termW, height: termH }, renderLayout);
    } else {
      drawTerminalFallback(ctx, screen, renderLayout);
    }
    drawGridLabelOverlay(ctx, renderLayout, prepared.options);

    const needsDownscale =
      !skipDownscale && (renderLayout.outerW !== out.width || renderLayout.outerH !== out.height);
    const canvas = needsDownscale ? downscaleCanvas(out, renderLayout.outerW, renderLayout.outerH) : out;
    return { canvas, layout: renderLayout };
  } finally {
    prepared.dispose();
  }
}

export async function composeExportPNG({ screen, replayBytes, opts, backgroundFile, viewerMetrics }) {
  const { canvas } = await composeExportRasterCanvas({
    screen,
    replayBytes,
    opts,
    backgroundFile,
    viewerMetrics,
  });
  return await new Promise((resolve, reject) => {
    canvas.toBlob((b) => (b ? resolve(b) : reject(new Error("png encode failed"))), "image/png");
  });
}

export async function composeExport({ screen, replayBytes, opts, backgroundFile, viewerMetrics }) {
  const options = validateExportOptions(opts);
  if (options.format === FORMAT_SVG) {
    return composeExportSVG({ screen, replayBytes, opts: options, backgroundFile, viewerMetrics });
  }
  return composeExportPNG({ screen, replayBytes, opts: options, backgroundFile, viewerMetrics });
}

export async function composeExportSVG({ screen, replayBytes, opts, backgroundFile, viewerMetrics }) {
  const options = validateExportOptions({ ...opts, format: FORMAT_SVG });
  const { canvas, layout } = await composeExportRasterCanvas({
    screen,
    replayBytes,
    opts: options,
    backgroundFile,
    viewerMetrics,
    // Keep supersampled pixels; SVG viewport stays at logical 1x size so viewers scale down sharply.
    skipDownscale: true,
  });
  const embedFill = svgEmbedBackgroundFill(options);
  const embedCanvas = embedFill ? flattenCanvasForSvgEmbed(canvas, embedFill) : canvas;
  const logicalW = layout.outerW;
  const logicalH = layout.outerH;
  const dataUrl = await canvasToDataUrl(embedCanvas, "image/png");
  const svg = wrapRasterSvg(dataUrl, logicalW, logicalH);
  return new Blob([svg], { type: "image/svg+xml" });
}

export function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
