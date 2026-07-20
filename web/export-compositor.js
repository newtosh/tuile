import {
  BACKGROUND_CUSTOM,
  BACKGROUND_PRESET,
  BACKGROUND_PRESETS,
  BACKGROUND_TRANSPARENT,
  CHROME_MINIMAL,
  CHROME_OS_WIREFRAME,
  FORMAT_PNG,
  FORMAT_SVG,
  VIEWER_FRAME,
  chromeInnerGap,
  chromePadding,
  exportScales,
  gridLabelMetrics,
  titleBarHeight,
  validateExportOptions,
  viewerFrameMetrics,
} from "./export-options.js";

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

  if (opts.chrome_preset === CHROME_OS_WIREFRAME) {
    const pad = chromePadding() * renderScale;
    const title = titleBarHeight(CHROME_OS_WIREFRAME) * renderScale;
    const inner = chromeInnerGap() * renderScale;
    const renderOuterW = termW + pad * 2;
    const renderOuterH = pad + title + inner + termH + pad;
    return {
      ...base,
      chrome: CHROME_OS_WIREFRAME,
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

  const frame = viewerFrameMetrics(renderScale);
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
  if (opts.chrome_preset === CHROME_OS_WIREFRAME) {
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
  const frame = viewerFrameMetrics(layout.renderScale);
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

function drawViewerFrame(ctx, layout) {
  const s = layout.renderScale ?? layout.scale;
  const w = layout.frameW;
  const h = layout.frameH;

  ctx.save();
  ctx.shadowColor = "rgba(0, 0, 0, 0.28)";
  ctx.shadowBlur = 32 * s;
  ctx.shadowOffsetY = 12 * s;
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
  ctx.fillStyle = VIEWER_FRAME.labelBg;
  ctx.fill();
  ctx.strokeStyle = VIEWER_FRAME.labelBorder;
  ctx.lineWidth = 1;
  ctx.stroke();
  ctx.fillStyle = VIEWER_FRAME.labelText;
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

function drawWireframeChrome(ctx, layout, opts) {
  const s = layout.scale;
  const inset = layout.pad;
  const { w, h } = canvasSize(layout);

  ctx.fillStyle = FRAME_FILL;
  ctx.fillRect(0, 0, w, h);
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
  if (opts.chrome_preset === CHROME_OS_WIREFRAME) {
    drawWireframeChrome(ctx, layout, opts);
    return;
  }
  drawViewerFrame(ctx, layout);
}

function drawGridLabelOverlay(ctx, layout, opts) {
  if (opts.chrome_preset === CHROME_OS_WIREFRAME || !opts.show_grid_size) {
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
  const term = new Terminal({
    cols: viewerMetrics?.cols || layout.cols,
    rows: viewerMetrics?.rows || layout.rows,
    fontFamily: options.font_family,
    fontSize: terminalFontPx,
    lineHeight: 1,
    customGlyphs: true,
    drawBoldTextInBrightColors: true,
    scrollback: 0,
    convertEol: true,
    allowProposedApi: true,
    theme: {
      background: TERM_BG,
      foreground: "#e4e4e4",
      black: TERM_BG,
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
  });
  term.open(host);

  let webglAddon = null;
  if (window.Unicode11Addon) {
    const unicode11Addon = new Unicode11Addon.Unicode11Addon();
    term.loadAddon(unicode11Addon);
    term.unicode.activeVersion = "11";
  }
  if (window.WebglAddon) {
    try {
      webglAddon = new WebglAddon.WebglAddon();
      term.loadAddon(webglAddon);
    } catch {
      webglAddon = null;
    }
  }

  return { term, webglAddon };
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
  ctx.fillStyle = TERM_BG;
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

  const { term, webglAddon } = loadExportTerminal(host, layout, options, viewerMetrics);
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
    await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));

    const termCanvas = compositeTerminalCanvases(host);
    const measured = measureExportTerminal(term, termCanvas);
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
    webglAddon?.dispose();
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
  const isWireframe = options.chrome_preset === CHROME_OS_WIREFRAME;
  let svg = `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" width="${layout.outerW}" height="${layout.outerH}" viewBox="0 0 ${layout.renderOuterW} ${layout.renderOuterH}">`;

  if (options.background_mode === BACKGROUND_PRESET) {
    const spec = BACKGROUND_PRESETS[options.background_preset] || BACKGROUND_PRESETS.slate;
    if (spec.kind === "solid") {
      svg += `<rect width="100%" height="100%" fill="${spec.color}"/>`;
    } else {
      svg += `<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="${spec.start}"/><stop offset="100%" stop-color="${spec.end}"/></linearGradient></defs><rect width="100%" height="100%" fill="url(#g)"/>`;
    }
  }

  if (isWireframe) {
    const s = layout.renderScale;
    const stroke = STROKE;
    const dash = `${5 * s} ${4 * s}`;
    svg += `<rect width="${layout.renderOuterW}" height="${layout.renderOuterH}" fill="${FRAME_FILL}" stroke="${stroke}" stroke-width="${2 * s}" stroke-dasharray="${dash}"/>`;
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
    svg += `<rect x="0" y="0" width="${layout.frameW}" height="${layout.frameH}" rx="${layout.radius}" fill="${layout.frameBg}" stroke="${layout.border}" stroke-width="1"/>`;
  }

  svg += `<g transform="translate(${layout.termX},${layout.termY})"><rect width="${layout.termW}" height="${layout.termH}" fill="${TERM_BG}"/>`;
  const termFontSize = layout.fontPx * layout.renderScale;
  lines.forEach((line, y) => {
    svg += `<text x="4" y="${(y + 1) * layout.cellH - 4}" fill="#e4e4e4" font-family="monospace" font-size="${termFontSize}">${escapeXml(line)}</text>`;
  });
  svg += `</g>`;

  if (!isWireframe && options.show_grid_size) {
    const label = `${layout.cols}×${layout.rows}`;
    const badge = gridLabelMetrics(layout.renderScale);
    const anchorX = layout.frameW;
    const anchorY = layout.frameH;
    const boxW = label.length * badge.fontSize * 0.62 + badge.padX * 2;
    const boxH = badge.fontSize + badge.padY * 2;
    const lx = anchorX - boxW - badge.offsetX;
    const ly = anchorY - boxH - badge.offsetY;
    svg += `<rect x="${lx}" y="${ly}" width="${boxW}" height="${boxH}" rx="${badge.radius}" fill="${VIEWER_FRAME.labelBg}" stroke="${VIEWER_FRAME.labelBorder}" stroke-width="1"/>`;
    svg += `<text x="${lx + badge.padX}" y="${ly + badge.padY + badge.fontSize * 0.88}" fill="${VIEWER_FRAME.labelText}" font-family="JetBrains Mono, ui-monospace, monospace" font-size="${badge.fontSize}" font-weight="500">${escapeXml(label)}</text>`;
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
