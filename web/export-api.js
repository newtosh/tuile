import { FORMAT_SVG, validateExportOptions } from "./export-options.js";

export function buildServerExportPayload(opts, viewerMetrics = null) {
  const options = validateExportOptions({ ...opts, format: FORMAT_SVG });
  const payload = {
    chrome_preset: options.chrome_preset,
    chrome_os_style: options.chrome_os_style,
    background_mode: options.background_mode,
    background_preset: options.background_preset,
    scale: options.scale,
    format: FORMAT_SVG,
    font_family: viewerMetrics?.fontFamily || options.font_family,
    font_size_px: viewerMetrics?.fontSizePx || options.font_size_px,
    theme: options.theme,
    title: options.title,
    show_grid_size: options.show_grid_size,
  };
  // SVG is vector output: keep cell geometry derived from font_size_px + cols/rows.
  // Viewer term pixel metrics mismatch font size and cause overlapping glyphs.
  return payload;
}

export async function requestServerExport({
  sessionId,
  token,
  opts,
  backgroundFile = null,
  viewerMetrics = null,
  apiURL,
}) {
  if (!sessionId || !token) {
    throw new Error("session required for export");
  }
  const payload = buildServerExportPayload(opts, viewerMetrics);
  const url = apiURL(`/v1/sessions/${sessionId}/export`);
  const headers = { Authorization: `Bearer ${token}` };

  let res;
  if (payload.background_mode === "custom" && backgroundFile) {
    const form = new FormData();
    for (const [key, value] of Object.entries(payload)) {
      if (value === undefined || value === null) {
        continue;
      }
      form.append(key, typeof value === "boolean" ? (value ? "1" : "0") : String(value));
    }
    form.append("background_image", backgroundFile);
    res = await fetch(url, { method: "POST", headers, body: form });
  } else {
    res = await fetch(url, {
      method: "POST",
      headers: { ...headers, "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  }

  if (!res.ok) {
    let detail = "";
    try {
      detail = await res.text();
    } catch {
      // ignore
    }
    throw new Error(`export failed (${res.status})${detail ? `: ${detail}` : ""}`);
  }
  return await res.blob();
}
