import { mountIcon } from "./icons.js";
import { sessions as $sessions, activeSessionId as $activeSessionId, uiStatus, uiBadge } from "./state.js";
import {
  ACK_STORAGE_KEY,
  loadAckMap,
  pruneClientSessionState,
  saveAckMap,
} from "./session-state.js";

const params = new URLSearchParams(window.location.search);
const BOOTSTRAP_KEY = "tuile_bootstrap";
const SESSION_SORT_KEY = "tuile_session_sort";
const SESSION_ACK_KEY = ACK_STORAGE_KEY;
const SESSION_INACTIVE_MINS_KEY = "tuile_session_inactive_mins";
const SESSION_SORT_VALUES = [
  "created-desc",
  "created-asc",
  "label-asc",
  "label-desc",
  "id-asc",
  "duration-desc",
  "duration-asc",
];
const POLL_MS = 2000;
const REPLAY_RESET = "\x1b[?1049l\x1b[0m\x1b[2J\x1b[H";
const LOAD_TOTAL_TIMEOUT_MS = 25000;
const LOAD_ATTACH_TIMEOUT_MS = 12000;
const LOAD_WS_CONNECT_TIMEOUT_MS = 10000;
const LOAD_SNAPSHOT_TIMEOUT_MS = 8000;
const REFRESH_SPIN_MIN_MS = 450;
const REFRESH_BUTTON_TIMEOUT_MS = 8000;
const sessionCache = new Map();

const badge = document.getElementById("mode-badge");
const statusBar = document.getElementById("status-bar");
const statusMessage = document.getElementById("status-message");
const zoomOutBtn = document.getElementById("zoom-out");
const zoomInBtn = document.getElementById("zoom-in");
const zoomResetBtn = document.getElementById("zoom-reset");
const takeoverBtn = document.getElementById("takeover");
const releaseBtn = document.getElementById("release");
const reconnectBtn = document.getElementById("reconnect");
const fontSelect = document.getElementById("font-family");
const fontSizeSelect = document.getElementById("font-size");
const webglToggle = document.getElementById("webgl-renderer");
const settingsToggle = document.getElementById("settings-toggle");
const settingsMenu = document.getElementById("settings-menu");
const terminalWrap = document.getElementById("terminal-wrap");
const terminalGridFrame = document.getElementById("terminal-grid-frame");
const gridFrameLabel = document.getElementById("terminal-grid-label");
const terminalPlaceholder = document.getElementById("terminal-placeholder");
const terminalLoading = document.getElementById("terminal-loading");
const loadingSpinner = document.getElementById("loading-spinner");
const loadingMessage = document.getElementById("loading-message");
const loadingRetry = document.getElementById("loading-retry");
const sessionList = document.getElementById("session-list");
const sessionEmpty = document.getElementById("session-empty");
const sessionSortSelect = document.getElementById("session-sort");
const sessionInactiveMins = document.getElementById("session-inactive-mins");
const refreshSessionsBtn = document.getElementById("refresh-sessions");
const bootstrapForm = document.getElementById("bootstrap-form");
const bootstrapInput = document.getElementById("bootstrap-secret");

let bootstrapSecret = localStorage.getItem(BOOTSTRAP_KEY) || "";
let sessionId = params.get("session");
let token = params.get("token");
let controlling = false;
let ws = null;
let resizeTimer = null;
let pollTimer = null;
const DEFAULT_PTY_COLS = 120;
const DEFAULT_PTY_ROWS = 36;
let ptyCols = DEFAULT_PTY_COLS;
let ptyRows = DEFAULT_PTY_ROWS;
let knownSessions = [];
let attaching = false;
let sessionLoading = false;
let loadingTimeoutTimer = null;
let wsConnectTimeoutTimer = null;
let loadAttempt = 0;
let layoutTimer = null;
let awaitingInitialSync = false;
let syncFallbackTimer = null;
let wsWriteChain = Promise.resolve();
const FONT_SIZE_KEY = "tuile_font_size";
const DEFAULT_FONT_SIZE = 20;
let observeBaseFont = DEFAULT_FONT_SIZE;
const OBSERVE_FONT_MIN = 12;
const OBSERVE_FONT_MAX = 64;
const OBSERVE_VIEW_INSET = 4;
const GRID_FRAME_PAD = 14;
const ZOOM_KEY = "tuile_zoom";
const ZOOM_MIN = 0.5;
const ZOOM_MAX = 1.5;
const ZOOM_STEP = 0.05;
const WEBGL_KEY = "tuile_webgl";
const LEGACY_LIGATURES_KEY = "tuile_ligatures";
let observeZoom = clampZoom(parseFloat(localStorage.getItem(ZOOM_KEY)) || 1);
let fontSizeMode = localStorage.getItem(FONT_SIZE_KEY) || "20";

if (params.get("bootstrap")) {
  bootstrapSecret = params.get("bootstrap");
  localStorage.setItem(BOOTSTRAP_KEY, bootstrapSecret);
  params.delete("bootstrap");
  const next = `${window.location.pathname}${params.toString() ? `?${params}` : ""}`;
  window.history.replaceState(null, "", next);
}

bootstrapInput.value = bootstrapSecret;
fontSizeSelect.value = fontSizeMode;
observeBaseFont = fontSizeMode === "auto" ? DEFAULT_FONT_SIZE : parseInt(fontSizeMode, 10) || DEFAULT_FONT_SIZE;

sessionSortSelect.value = loadSessionSort();
sessionInactiveMins.value = String(getInactiveMins());

uiStatus.subscribe((text) => {
  statusMessage.textContent = text;
});

uiBadge.subscribe(({ text, className }) => {
  badge.textContent = text;
  badge.className = `badge ${className}`.trim();
});

$sessions.subscribe((list) => {
  renderSessionList(list);
});

mountIcon(document.getElementById("settings-toggle-icon"), "settings", { size: 18 });
mountIcon(document.getElementById("webgl-info-icon"), "circle-help", { size: 14 });
mountIcon(document.getElementById("refresh-sessions-icon"), "refresh-cw", { size: 16 });
mountIcon(document.getElementById("bootstrap-save-icon"), "save", { size: 16 });
mountIcon(document.getElementById("zoom-out-icon"), "zoom-out", { size: 14 });
mountIcon(document.getElementById("zoom-in-icon"), "zoom-in", { size: 14 });
for (const slot of document.querySelectorAll("[data-icon]")) {
  mountIcon(slot, slot.dataset.icon, { size: 16 });
}

function setSettingsOpen(open) {
  settingsMenu.hidden = !open;
  settingsToggle.setAttribute("aria-expanded", String(open));
}

settingsToggle.addEventListener("click", (ev) => {
  ev.stopPropagation();
  setSettingsOpen(settingsMenu.hidden);
});

document.addEventListener("click", (ev) => {
  if (!settingsMenu.hidden && !ev.target.closest(".settings-wrap")) {
    setSettingsOpen(false);
  }
});

settingsMenu.addEventListener("click", (ev) => {
  ev.stopPropagation();
});

document.addEventListener("keydown", (ev) => {
  if (ev.key === "Escape" && !settingsMenu.hidden) {
    setSettingsOpen(false);
    settingsToggle.focus();
  }
});

const term = new Terminal({
  cursorBlink: true,
  fontSize: observeBaseFont,
  lineHeight: 1,
  fontFamily: fontSelect.value,
  scrollback: 5000,
  customGlyphs: true,
  drawBoldTextInBrightColors: true,
  minimumContrastRatio: 1,
  allowTransparency: false,
  theme: {
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
  allowProposedApi: true,
});
const fitAddon = new FitAddon.FitAddon();
const unicode11Addon = new Unicode11Addon.Unicode11Addon();
let webglAddon = null;

term.loadAddon(fitAddon);
term.loadAddon(unicode11Addon);
term.unicode.activeVersion = "11";
term.open(terminalWrap);

function readWebGLPref() {
  if (localStorage.getItem(WEBGL_KEY) !== null) {
    return localStorage.getItem(WEBGL_KEY) === "1";
  }
  if (localStorage.getItem(LEGACY_LIGATURES_KEY) !== null) {
    return localStorage.getItem(LEGACY_LIGATURES_KEY) === "1";
  }
  return true;
}

function setWebGLClass(enabled) {
  terminalWrap.classList.toggle("webgl-on", enabled);
}

function updateWebGLControl() {
  webglToggle.disabled = false;
  if (localStorage.getItem(WEBGL_KEY) === null && localStorage.getItem(LEGACY_LIGATURES_KEY) === null) {
    localStorage.setItem(WEBGL_KEY, "0");
  }
  webglToggle.checked = readWebGLPref();
  webglToggle.setAttribute("aria-checked", String(webglToggle.checked));
  setWebGLClass(webglToggle.checked);
  setWebGLRenderer(webglToggle.checked);
}

function setWebGLRenderer(enabled) {
  if (enabled && !webglAddon && window.WebglAddon) {
    try {
      webglAddon = new WebglAddon.WebglAddon();
      term.loadAddon(webglAddon);
      webglAddon.onContextLoss(() => {
        webglAddon?.dispose();
        webglAddon = null;
        if (webglToggle.checked) {
          setStatus("WebGL context lost — enhanced rendering paused.");
        }
      });
      term.refresh(0, term.rows - 1);
    } catch (err) {
      setStatus(`WebGL renderer unavailable: ${err.message}`);
      webglToggle.checked = false;
      webglToggle.setAttribute("aria-checked", "false");
      setWebGLClass(false);
    }
    return;
  }
  if (!enabled && webglAddon) {
    webglAddon.dispose();
    webglAddon = null;
    term.refresh(0, term.rows - 1);
  }
}

updateWebGLControl();

function clampZoom(value) {
  return Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, Math.round(value / ZOOM_STEP) * ZOOM_STEP));
}

function setObserveZoom(value, { persist = true, relayout = true } = {}) {
  observeZoom = clampZoom(value);
  if (persist) {
    localStorage.setItem(ZOOM_KEY, String(observeZoom));
  }
  updateZoomControl();
  if (relayout && !controlling) {
    scheduleTerminalLayout();
  }
}

function updateZoomControl() {
  const pct = Math.round(observeZoom * 100);
  zoomResetBtn.textContent = `${pct}%`;
  zoomResetBtn.classList.toggle("is-custom", Math.abs(observeZoom - 1) > 0.001);
  zoomOutBtn.disabled = observeZoom <= ZOOM_MIN + 0.001;
  zoomInBtn.disabled = observeZoom >= ZOOM_MAX - 0.001;
}

function setStatus(text) {
  uiStatus.set(text);
}

function setControlChrome(isControl) {
  statusBar.classList.toggle("is-control", isControl);
}

function setMode(mode, className = "") {
  uiBadge.set({ text: mode, className: className || mode });
}

function apiURL(path) {
  return `${window.location.origin}${path}`;
}

function authHeaders() {
  return {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
  };
}

function bootstrapHeaders() {
  return {
    Authorization: `Bearer ${bootstrapSecret}`,
    "Content-Type": "application/json",
  };
}

function shortID(id) {
  return id ? `${id.slice(0, 8)}…` : "";
}

function basename(path) {
  if (!path) {
    return "";
  }
  const parts = path.split("/").filter(Boolean);
  return parts.length ? parts[parts.length - 1] : path;
}

function updateURL() {
  const next = new URLSearchParams();
  if (sessionId) {
    next.set("session", sessionId);
  }
  if (token) {
    next.set("token", token);
  }
  const qs = next.toString();
  window.history.replaceState(null, "", `${window.location.pathname}${qs ? `?${qs}` : ""}`);
}

function showPlaceholder(show) {
  terminalPlaceholder.hidden = !show;
  terminalWrap.style.visibility = show ? "hidden" : "visible";
  if (show) {
    setSessionLoading(false);
  }
}

function isSessionLoading() {
  return attaching || awaitingInitialSync || sessionLoading;
}

function updateSessionLoadingState() {
  const loading = isSessionLoading() && !terminalLoading.classList.contains("is-error");
  for (const btn of sessionList.querySelectorAll(".session-item")) {
    btn.classList.toggle("loading", loading && btn.dataset.sessionId === sessionId);
  }
}

function clearLoadTimeouts() {
  clearTimeout(loadingTimeoutTimer);
  clearTimeout(wsConnectTimeoutTimer);
  loadingTimeoutTimer = null;
  wsConnectTimeoutTimer = null;
}

function resetLoadingChrome() {
  terminalLoading.classList.remove("is-error");
  loadingSpinner.hidden = false;
  loadingMessage.classList.remove("is-error");
  loadingRetry.hidden = true;
}

function failSessionLoad(message, { retry = true } = {}) {
  clearLoadTimeouts();
  clearSyncPoll();
  clearTimeout(syncFallbackTimer);
  syncFallbackTimer = null;
  awaitingInitialSync = false;
  attaching = false;
  terminalWrap.classList.remove("switching");
  setMode("error", "error");
  setStatus(message);
  sessionLoading = true;
  terminalLoading.hidden = false;
  terminalLoading.setAttribute("aria-busy", "false");
  terminalLoading.classList.add("is-error");
  loadingSpinner.hidden = true;
  loadingMessage.textContent = message;
  loadingMessage.classList.add("is-error");
  loadingRetry.hidden = !retry;
  terminalWrap.classList.add("loading");
  updateSessionLoadingState();
  if (ws) {
    const sock = ws;
    ws = null;
    sock.onclose = null;
    sock.onerror = null;
    sock.close();
  }
}

function setSessionLoading(loading, message = "Loading session…") {
  if (loading) {
    resetLoadingChrome();
    sessionLoading = true;
    const attempt = ++loadAttempt;
    terminalLoading.hidden = false;
    terminalLoading.setAttribute("aria-busy", "true");
    loadingMessage.textContent = message;
    terminalWrap.classList.add("loading");
    clearLoadTimeouts();
    loadingTimeoutTimer = setTimeout(() => {
      if (attempt !== loadAttempt || !sessionLoading) {
        return;
      }
      failSessionLoad(
        "Session load timed out. The server may be unreachable or the session ended.",
        { retry: true }
      );
    }, LOAD_TOTAL_TIMEOUT_MS);
  } else {
    loadAttempt += 1;
    clearLoadTimeouts();
    sessionLoading = false;
    terminalLoading.hidden = true;
    terminalLoading.setAttribute("aria-busy", "false");
    resetLoadingChrome();
    terminalWrap.classList.remove("loading");
  }
  updateSessionLoadingState();
}

function updateSessionListActive() {
  for (const btn of sessionList.querySelectorAll(".session-item")) {
    btn.classList.toggle("active", btn.dataset.sessionId === sessionId);
  }
  $activeSessionId.set(sessionId);
}

function syncSessions(list) {
  knownSessions = list;
  pruneClientSessionState({ cache: sessionCache, sessions: list });
  if (params.get("debug") === "memory") {
    const ackCount = Object.keys(loadAckMap()).length;
    console.debug(
      `[tuile] memory debug: sessions=${list.length} cache=${sessionCache.size} ack=${ackCount}`
    );
  }
  $sessions.set(list);
}

function isDefaultDims(cols, rows) {
  return cols === DEFAULT_PTY_COLS && rows === DEFAULT_PTY_ROWS;
}

function sessionLabel(sess) {
  const dimLabel = isDefaultDims(sess.cols, sess.rows)
    ? `${sess.cols}×${sess.rows}`
    : `${sess.cols}×${sess.rows} (resized)`;
  const parts = [shortID(sess.session_id), dimLabel];
  if (sess.cli) {
    parts.push(sess.cli);
  }
  parts.push(sess.controller);
  return parts.join(" · ");
}

function displayLabel(sess) {
  return sess.cli || basename(sess.workspace);
}

function parseSessionTime(iso) {
  const t = Date.parse(iso || "");
  return Number.isFinite(t) ? t : 0;
}

function loadSessionSort() {
  const raw = localStorage.getItem(SESSION_SORT_KEY);
  return SESSION_SORT_VALUES.includes(raw) ? raw : "created-desc";
}

function saveSessionSort(value) {
  localStorage.setItem(SESSION_SORT_KEY, value);
}

function getInactiveMins() {
  const stored = parseInt(localStorage.getItem(SESSION_INACTIVE_MINS_KEY) || "15", 10);
  if (!Number.isFinite(stored) || stored < 1) {
    return 15;
  }
  return Math.min(stored, 1440);
}

function saveInactiveMins(value) {
  localStorage.setItem(SESSION_INACTIVE_MINS_KEY, String(value));
}

function acknowledgeSession(id) {
  if (!id) {
    return;
  }
  const map = loadAckMap();
  map[id] = new Date().toISOString();
  saveAckMap(map);
}

function computeSessionStatus(sess) {
  const last = parseSessionTime(sess.last_meaningful_activity_at);
  if (!last) {
    return "idle";
  }
  const inactiveMs = getInactiveMins() * 60 * 1000;
  if (Date.now() - last > inactiveMs) {
    return "inactive";
  }
  const ack = parseSessionTime(loadAckMap()[sess.session_id]);
  if (last > ack) {
    return "active";
  }
  return "idle";
}

function sortSessions(list, sortKey = loadSessionSort()) {
  const sorted = [...list];
  const byCreated = (a, b) => parseSessionTime(a.created_at) - parseSessionTime(b.created_at);
  const byLabel = (a, b) => displayLabel(a).localeCompare(displayLabel(b), undefined, { sensitivity: "base" });
  const byID = (a, b) => a.session_id.localeCompare(b.session_id);
  const byDuration = (a, b) => {
    const now = Date.now();
    return now - parseSessionTime(a.created_at) - (now - parseSessionTime(b.created_at));
  };

  switch (sortKey) {
    case "created-asc":
      sorted.sort(byCreated);
      break;
    case "label-asc":
      sorted.sort(byLabel);
      break;
    case "label-desc":
      sorted.sort((a, b) => byLabel(b, a));
      break;
    case "id-asc":
      sorted.sort(byID);
      break;
    case "duration-desc":
      sorted.sort((a, b) => byDuration(b, a));
      break;
    case "duration-asc":
      sorted.sort(byDuration);
      break;
    case "created-desc":
    default:
      sorted.sort((a, b) => byCreated(b, a));
      break;
  }
  return sorted;
}

function sessionRowSignature(sess) {
  return [
    sess.session_id,
    sess.workspace,
    sess.cli || "",
    sess.cols,
    sess.rows,
    sess.controller,
    sess.created_at || "",
    sess.last_meaningful_activity_at || "",
    computeSessionStatus(sess),
    sessionId === sess.session_id ? "active" : "",
    sessionLoading && sessionId === sess.session_id ? "loading" : "",
  ].join("|");
}

function updateSessionRow(li, sess) {
  const btn = li.querySelector(".session-item");
  const workspace = li.querySelector(".workspace");
  const meta = li.querySelector(".meta");
  const status = li.querySelector(".session-status");
  if (!btn || !workspace || !meta || !status) {
    return;
  }

  workspace.textContent = displayLabel(sess);
  workspace.title = sess.workspace;
  meta.textContent = sessionLabel(sess);

  const statusState = computeSessionStatus(sess);
  status.className = `session-status ${statusState}`;
  status.setAttribute(
    "aria-label",
    statusState === "active"
      ? "Recent activity"
      : statusState === "inactive"
        ? "Inactive session"
        : "Idle session"
  );

  btn.className = `session-item${sess.session_id === sessionId ? " active" : ""}${
    sessionLoading && sess.session_id === sessionId ? " loading" : ""
  }`;
}

function createSessionRow(sess) {
  const li = document.createElement("li");
  li.className = "session-row";
  li.dataset.sessionId = sess.session_id;

  const btn = document.createElement("button");
  btn.type = "button";
  btn.dataset.sessionId = sess.session_id;
  btn.className = "session-item";

  const labelRow = document.createElement("span");
  labelRow.className = "session-label-row";

  const status = document.createElement("span");
  status.className = "session-status";
  status.setAttribute("aria-hidden", "true");

  const workspace = document.createElement("span");
  workspace.className = "workspace";

  labelRow.appendChild(status);
  labelRow.appendChild(workspace);

  const meta = document.createElement("span");
  meta.className = "meta";

  btn.appendChild(labelRow);
  btn.appendChild(meta);
  btn.addEventListener("click", () => {
    acknowledgeSession(sess.session_id);
    const current = knownSessions.find((s) => s.session_id === sess.session_id) || sess;
    updateSessionRow(li, current);
    if (sess.session_id !== sessionId) {
      syncPtyDimensionsFromSession(current);
      attachToSession(sess.session_id);
    }
  });

  const closeBtn = document.createElement("button");
  closeBtn.type = "button";
  closeBtn.className = "session-close icon-btn";
  closeBtn.title = "Close session";
  closeBtn.setAttribute("aria-label", `Close session ${shortID(sess.session_id)}`);
  closeBtn.innerHTML = '<span class="icon-slot"></span>';
  mountIcon(closeBtn.querySelector(".icon-slot"), "trash-2", { size: 14 });
  closeBtn.addEventListener("click", async (ev) => {
    ev.stopPropagation();
    try {
      await closeSession(sess.session_id);
      await refreshSessions({ autoAttach: false });
    } catch (err) {
      setStatus(`Close failed: ${err.message}`);
    }
  });

  li.appendChild(btn);
  li.appendChild(closeBtn);
  updateSessionRow(li, sess);
  return li;
}

function syncPtyDimensionsFromSession(sess) {
  if (!sess) {
    return;
  }
  if (sess.cols) {
    ptyCols = sess.cols;
  }
  if (sess.rows) {
    ptyRows = sess.rows;
  }
}

function syncPtyDimensionsFromList(id) {
  syncPtyDimensionsFromSession(knownSessions.find((s) => s.session_id === id));
}

async function closeSession(id) {
  if (!bootstrapSecret || !id) {
    return;
  }
  const res = await fetch(apiURL(`/v1/sessions/${id}`), {
    method: "DELETE",
    headers: bootstrapHeaders(),
  });
  if (!res.ok && res.status !== 404) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `close failed: ${res.status}`);
  }
  sessionCache.delete(id);
  if (sessionId === id) {
    disconnectWS();
    sessionId = null;
    token = null;
    updateURL();
    showPlaceholder(true);
    setMode("waiting");
    setStatus("Session closed — select another or wait for a new one.");
  }
}

function renderSessionList(list = knownSessions) {
  const sorted = sortSessions(list);
  const nextIds = sorted.map((s) => s.session_id);
  const existingRows = [...sessionList.querySelectorAll(".session-row")];
  const existingIds = existingRows.map((li) => li.dataset.sessionId);
  const signatures = sorted.map(sessionRowSignature);
  const existingSignatures = existingIds.map((id) => {
    const sess = sorted.find((s) => s.session_id === id);
    return sess ? sessionRowSignature(sess) : "";
  });

  const orderUnchanged =
    existingIds.length === nextIds.length && existingIds.every((id, i) => id === nextIds[i]);
  const contentUnchanged =
    orderUnchanged && signatures.every((sig, i) => sig === existingSignatures[i]);

  if (contentUnchanged) {
    updateSessionListActive();
    sessionEmpty.hidden = sorted.length > 0 || !bootstrapSecret;
    updateSessionLoadingState();
    return;
  }

  const rowMap = new Map();
  for (const li of existingRows) {
    rowMap.set(li.dataset.sessionId, li);
  }

  for (const sess of sorted) {
    let li = rowMap.get(sess.session_id);
    if (!li) {
      li = createSessionRow(sess);
      sessionList.appendChild(li);
      rowMap.set(sess.session_id, li);
      continue;
    }
    updateSessionRow(li, sess);
  }

  for (const li of [...sessionList.querySelectorAll(".session-row")]) {
    if (!nextIds.includes(li.dataset.sessionId)) {
      li.remove();
    }
  }

  for (const sess of sorted) {
    const li = rowMap.get(sess.session_id);
    if (li) {
      sessionList.appendChild(li);
    }
  }

  sessionEmpty.hidden = sorted.length > 0 || !bootstrapSecret;
  $activeSessionId.set(sessionId);
  updateSessionLoadingState();
}

function clearTerminalTransform() {
  const termEl = terminalWrap.querySelector(".xterm");
  if (!termEl) {
    hideGridFrame();
    return;
  }
  termEl.style.transform = "";
  termEl.style.marginLeft = "";
  termEl.style.marginTop = "";
  clearObserveGridInlineSizes();
  hideGridFrame();
}

function terminalElement() {
  return terminalWrap.querySelector(".xterm");
}

function gridCellSize() {
  const cell = term._core?._renderService?.dimensions?.css?.cell;
  if (cell?.width && cell?.height) {
    return { width: cell.width, height: cell.height };
  }
  return null;
}

function isIntegerCellSize(cell) {
  return (
    Math.abs(cell.width - Math.round(cell.width)) < 0.01 &&
    Math.abs(cell.height - Math.round(cell.height)) < 0.01
  );
}

function measureTerminalGrid() {
  const css = term._core?._renderService?.dimensions?.css;
  if (css?.canvas?.width > 0 && css?.canvas?.height > 0) {
    return { width: css.canvas.width, height: css.canvas.height };
  }
  const cell = css?.cell ?? gridCellSize();
  if (cell?.width && cell?.height) {
    return {
      width: Math.ceil(term.cols * cell.width - 1e-9),
      height: Math.ceil(term.rows * cell.height - 1e-9),
    };
  }
  const termEl = terminalElement();
  const screen = termEl?.querySelector(".xterm-screen");
  return {
    width: screen?.offsetWidth ?? 0,
    height: screen?.offsetHeight ?? 0,
  };
}

function gridFitsTarget(grid, maxW, maxH) {
  return grid.width > 0 && grid.height > 0 && grid.width <= maxW && grid.height <= maxH;
}

function applyObserveFontSize(size) {
  term.options.fontSize = size;
  term.resize(ptyCols, ptyRows);
  term.refresh(0, term.rows - 1);
}

function refineObserveFontSize(size, targetW, targetH) {
  let next = size;
  applyObserveFontSize(next);
  while (next > OBSERVE_FONT_MIN) {
    const grid = measureTerminalGrid();
    const cell = gridCellSize();
    const fits = gridFitsTarget(grid, targetW, targetH);
    const integer = cell && isIntegerCellSize(cell);
    if (fits && integer) {
      break;
    }
    next -= 1;
    applyObserveFontSize(next);
  }
  return next;
}

function clearObserveGridInlineSizes() {
  const termEl = terminalElement();
  if (!termEl) {
    return;
  }
  termEl.style.width = "";
  termEl.style.height = "";
  termEl.style.maxWidth = "";
  for (const sel of [".xterm-viewport", ".xterm-screen", ".xterm-rows", "canvas"]) {
    const el = termEl.querySelector(sel);
    if (!el) {
      continue;
    }
    el.style.width = "";
    el.style.height = "";
    el.style.maxWidth = "";
  }
}

function hideGridFrame() {
  if (terminalGridFrame) {
    terminalGridFrame.hidden = true;
  }
  if (gridFrameLabel) {
    gridFrameLabel.hidden = true;
  }
}

function updateGridFrame() {
  if (!terminalGridFrame || controlling || !terminalWrap.classList.contains("observe-mode")) {
    hideGridFrame();
    return;
  }
  const grid = measureTerminalGrid();
  const termEl = terminalElement();
  const width = Math.max(grid.width, termEl?.offsetWidth ?? 0);
  const height = Math.max(grid.height, termEl?.offsetHeight ?? 0);
  if (!width || !height) {
    hideGridFrame();
    return;
  }
  const inset = OBSERVE_VIEW_INSET;
  const viewW = terminalWrap.clientWidth - inset * 2;
  const viewH = terminalWrap.clientHeight - inset * 2;
  const left = inset + Math.max(0, (viewW - width) / 2);
  const top = inset + Math.max(0, (viewH - height) / 2);
  const frameLeft = left - GRID_FRAME_PAD;
  const frameTop = top - GRID_FRAME_PAD;
  const frameWidth = width + GRID_FRAME_PAD * 2;
  const frameHeight = height + GRID_FRAME_PAD * 2;

  terminalGridFrame.hidden = false;
  terminalGridFrame.style.left = `${frameLeft}px`;
  terminalGridFrame.style.top = `${frameTop}px`;
  terminalGridFrame.style.width = `${frameWidth}px`;
  terminalGridFrame.style.height = `${frameHeight}px`;

  if (gridFrameLabel) {
    gridFrameLabel.hidden = false;
    gridFrameLabel.textContent = `${ptyCols}×${ptyRows}`;
    gridFrameLabel.style.left = `${frameLeft + frameWidth}px`;
    gridFrameLabel.style.top = `${frameTop + frameHeight}px`;
  }
}

function positionObserveTerminal() {
  const termEl = terminalElement();
  if (!termEl) {
    return;
  }
  const inset = OBSERVE_VIEW_INSET;
  const viewW = terminalWrap.clientWidth - inset * 2;
  const viewH = terminalWrap.clientHeight - inset * 2;

  termEl.style.transform = "";
  clearObserveGridInlineSizes();
  term.refresh(0, term.rows - 1);

  const grid = measureTerminalGrid();
  const width = Math.max(grid.width, termEl.offsetWidth);
  const height = Math.max(grid.height, termEl.offsetHeight);
  if (!width || !height) {
    return;
  }

  termEl.style.marginLeft = `${inset + Math.max(0, (viewW - width) / 2)}px`;
  termEl.style.marginTop = `${inset + Math.max(0, (viewH - height) / 2)}px`;
  updateGridFrame();
}

function maxFontForTarget(targetW, targetH, { cap } = {}) {
  let lo = OBSERVE_FONT_MIN;
  let hi = cap ?? OBSERVE_FONT_MAX;
  let best = OBSERVE_FONT_MIN;

  while (lo <= hi) {
    const mid = Math.floor((lo + hi) / 2);
    applyObserveFontSize(mid);
    const grid = measureTerminalGrid();
    if (gridFitsTarget(grid, targetW, targetH)) {
      best = mid;
      lo = mid + 1;
    } else {
      hi = mid - 1;
    }
  }

  return refineObserveFontSize(best, targetW, targetH);
}

function fitObserveLayout() {
  clearTerminalTransform();
  const inset = OBSERVE_VIEW_INSET;
  const viewW = Math.max(1, terminalWrap.clientWidth - inset * 2);
  const viewH = Math.max(1, terminalWrap.clientHeight - inset * 2);

  let targetW = viewW;
  let targetH = viewH;
  if (observeZoom < 1) {
    targetW = viewW * observeZoom;
    targetH = viewH * observeZoom;
  }

  let best;
  if (fontSizeMode === "auto") {
    best = maxFontForTarget(targetW, targetH);
    if (observeZoom > 1) {
      const boosted = Math.min(
        OBSERVE_FONT_MAX,
        maxFontForTarget(viewW, viewH, { cap: Math.floor(best * observeZoom) })
      );
      best = boosted;
      term.options.fontSize = best;
      term.resize(ptyCols, ptyRows);
    }
  } else {
    const preferred = parseInt(fontSizeMode, 10) || DEFAULT_FONT_SIZE;
    best = maxFontForTarget(targetW, targetH, { cap: preferred });
  }

  positionObserveTerminal();
  return { fontSize: best };
}

function formatObserveStatus({ fontSize }) {
  const parts = [`Observe — ${ptyCols}×${ptyRows}`, `${fontSize}px`];
  if (Math.abs(observeZoom - 1) > 0.001) {
    parts.push(`zoom ${Math.round(observeZoom * 100)}%`);
  }
  if (fontSizeMode !== "auto") {
    parts.push(fontSizeMode === String(fontSize) ? "fixed" : "clamped");
  }
  return parts.join(" · ");
}

function writeToTerminal(data) {
  return new Promise((resolve) => {
    term.write(data, resolve);
  });
}

function stripLeadingPartialEscape(bytes) {
  if (!bytes.length || bytes[0] === 0x1b || bytes[0] === 0x0a) {
    return bytes;
  }
  for (let i = 1; i < bytes.length; i++) {
    if (bytes[i] === 0x1b || bytes[i] === 0x0a) {
      return bytes.subarray(i);
    }
  }
  return new Uint8Array(0);
}

function decodeReplayB64(b64) {
  const raw = atob(b64);
  const bytes = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) {
    bytes[i] = raw.charCodeAt(i);
  }
  return stripLeadingPartialEscape(bytes);
}

function scaleTerminalObserve() {
  terminalWrap.classList.add("observe-mode");
  setControlChrome(false);
  const layout = fitObserveLayout();
  setStatus(formatObserveStatus(layout));
}

function applyTerminalLayout() {
  if (controlling) {
    terminalWrap.classList.remove("observe-mode");
    hideGridFrame();
    setControlChrome(true);
    clearTerminalTransform();
    setWebGLRenderer(readWebGLPref());
    term.options.fontSize = observeBaseFont;
    fitAddon.fit();
    scheduleHumanResize();
    setStatus(`Control — ${term.cols}×${term.rows}`);
    return;
  }
  // Observe mode uses the DOM renderer so zoom/font-fit stays aligned with the PTY grid.
  // WebGL loses its glyph atlas after observe layout (blank canvas at <100% zoom).
  setWebGLRenderer(false);
  term.resize(ptyCols, ptyRows);
  scaleTerminalObserve();
}

function scheduleTerminalLayout() {
  clearTimeout(layoutTimer);
  layoutTimer = setTimeout(() => {
    requestAnimationFrame(() => applyTerminalLayout());
  }, 80);
}

function finishInitialSync() {
  if (!awaitingInitialSync) {
    return;
  }
  awaitingInitialSync = false;
  clearSyncPoll();
  clearTimeout(syncFallbackTimer);
  syncFallbackTimer = null;
  terminalWrap.classList.remove("switching");
  setSessionLoading(false);
  updateSessionLoadingState();
  requestAnimationFrame(() => {
    requestAnimationFrame(() => applyTerminalLayout());
  });
}

async function fetchWithTimeout(url, options, timeoutMs) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(url, { ...options, signal: controller.signal });
  } finally {
    clearTimeout(timer);
  }
}

async function loadScreenSnapshot({ forceFinish = false } = {}) {
  const res = await fetchWithTimeout(
    apiURL(`/v1/sessions/${sessionId}/screen?replay=1`),
    { headers: authHeaders() },
    LOAD_SNAPSHOT_TIMEOUT_MS
  );
  if (!res.ok) {
    throw new Error(`screen fetch failed: ${res.status}`);
  }
  const body = await res.json();
  ptyCols = body.screen?.cols || ptyCols;
  ptyRows = body.screen?.rows || ptyRows;
  term.reset();
  term.resize(ptyCols, ptyRows);
  if (body.replay_b64) {
    const bytes = decodeReplayB64(body.replay_b64);
    await writeToTerminal(REPLAY_RESET);
    if (bytes.length) {
      await writeToTerminal(bytes);
    }
  } else {
    const lines = body.screen?.lines || [];
    if (lines.length > 0) {
      await writeToTerminal(lines.join("\r\n"));
    }
  }
  maybeFinishInitialSync({ force: forceFinish });
  return body.version;
}

let replayPrimed = false;
let syncPollTimer = null;

function hasVisibleTerminalContent() {
  const buffer = term.buffer.active;
  for (let row = 0; row < buffer.length; row++) {
    const line = buffer.getLine(row);
    if (!line) {
      continue;
    }
    for (let col = 0; col < line.length; col++) {
      const chars = line.getCell(col)?.getChars() ?? "";
      if (chars.trim()) {
        return true;
      }
    }
  }
  return false;
}

function clearSyncPoll() {
  clearInterval(syncPollTimer);
  syncPollTimer = null;
}

function maybeFinishInitialSync({ force = false } = {}) {
  if (!awaitingInitialSync) {
    return false;
  }
  if (force || hasVisibleTerminalContent()) {
    clearSyncPoll();
    finishInitialSync();
    return true;
  }
  return false;
}

function startSyncPoll() {
  clearSyncPoll();
  syncPollTimer = setInterval(() => {
    if (!awaitingInitialSync) {
      clearSyncPoll();
      return;
    }
    maybeFinishInitialSync();
  }, 250);
}

function writeWSChunk(data) {
  return new Promise((resolve) => {
    const write = () => {
      let chunk = data;
      if (chunk instanceof Uint8Array) {
        chunk = stripLeadingPartialEscape(chunk);
        if (!chunk.length) {
          resolve();
          return;
        }
      }
      term.write(chunk, () => {
        maybeFinishInitialSync();
        resolve();
      });
    };
    if (awaitingInitialSync && !replayPrimed) {
      replayPrimed = true;
      term.write(REPLAY_RESET, write);
      return;
    }
    write();
  });
}

function wsURL() {
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/v1/sessions/${sessionId}/ws?token=${encodeURIComponent(token)}`;
}

function disconnectWS() {
  clearTimeout(wsConnectTimeoutTimer);
  wsConnectTimeoutTimer = null;
  clearSyncPoll();
  if (ws) {
    ws.onclose = null;
    ws.close();
    ws = null;
  }
  clearTimeout(syncFallbackTimer);
  syncFallbackTimer = null;
  wsWriteChain = Promise.resolve();
  controlling = false;
  takeoverBtn.disabled = true;
  releaseBtn.disabled = true;
}

function connectWS() {
  if (!sessionId || !token) {
    showPlaceholder(true);
    setMode("idle");
    setStatus("Select a session from the list, or wait for one to appear.");
    return;
  }

  showPlaceholder(false);
  setMode("connecting");
  setStatus("Connecting WebSocket…");
  setSessionLoading(true, "Connecting to session…");
  takeoverBtn.disabled = true;
  releaseBtn.disabled = true;
  awaitingInitialSync = true;
  replayPrimed = false;
  disconnectWS();
  terminalWrap.classList.add("switching");
  updateSessionLoadingState();
  startSyncPoll();

  term.reset();
  term.resize(ptyCols, ptyRows);

  ws = new WebSocket(wsURL());
  ws.binaryType = "arraybuffer";
  const connectAttempt = loadAttempt;
  wsConnectTimeoutTimer = setTimeout(() => {
    if (connectAttempt !== loadAttempt || !ws || ws.readyState !== WebSocket.CONNECTING) {
      return;
    }
    const sock = ws;
    ws = null;
    sock.onclose = null;
    sock.onerror = null;
    sock.close();
    failSessionLoad("Could not connect to session — check that tuile is running.", { retry: true });
  }, LOAD_WS_CONNECT_TIMEOUT_MS);

  ws.onopen = () => {
    clearTimeout(wsConnectTimeoutTimer);
    wsConnectTimeoutTimer = null;
    setMode(controlling ? "control" : "observe", controlling ? "control" : "observe");
    takeoverBtn.disabled = controlling;
    releaseBtn.disabled = !controlling;
    setSessionLoading(true, "Syncing terminal…");
    setStatus("Syncing terminal…");
    updateSessionLoadingState();
    syncFallbackTimer = setTimeout(() => {
      if (!awaitingInitialSync) {
        return;
      }
      loadScreenSnapshot()
        .then(() => {
          if (!awaitingInitialSync) {
            return;
          }
          setSessionLoading(true, "Waiting for session output…");
          setStatus("Waiting for session output…");
          setMode(controlling ? "control" : "observe", controlling ? "control" : "observe");
        })
        .catch((err) => {
          if (!awaitingInitialSync) {
            return;
          }
          const msg =
            err.name === "AbortError"
              ? "Terminal sync timed out — no output received from the session."
              : `Could not load session output: ${err.message}`;
          failSessionLoad(msg, { retry: true });
        });
    }, 2000);
  };

  ws.onmessage = async (ev) => {
    let data;
    if (ev.data instanceof ArrayBuffer) {
      data = new Uint8Array(ev.data);
    } else if (ev.data instanceof Blob) {
      data = new Uint8Array(await ev.data.arrayBuffer());
    } else {
      data = ev.data;
    }
    wsWriteChain = wsWriteChain.then(() => writeWSChunk(data));
  };

  ws.onerror = () => {
    if (uiBadge.get().text === "error") {
      return;
    }
    failSessionLoad("WebSocket error — check token, Origin allowlist, and server.", { retry: true });
  };

  ws.onclose = () => {
    clearTimeout(wsConnectTimeoutTimer);
    wsConnectTimeoutTimer = null;
    if (uiBadge.get().text === "error") {
      takeoverBtn.disabled = true;
      releaseBtn.disabled = true;
      return;
    }
    if (awaitingInitialSync || sessionLoading) {
      failSessionLoad("Connection closed before the session finished loading.", { retry: true });
      return;
    }
    setMode("disconnected");
    setStatus("Disconnected. Reconnect or pick another session.");
    takeoverBtn.disabled = true;
    releaseBtn.disabled = true;
  };
}

async function attachToSession(id, { force = false } = {}) {
  if (!id || attaching) {
    return;
  }
  if (!force && id === sessionId && token && ws && ws.readyState === WebSocket.OPEN) {
    return;
  }

  attaching = true;
  setSessionLoading(true, "Attaching to session…");
  terminalWrap.classList.add("switching");
  updateSessionLoadingState();
  try {
    let cached = !force ? sessionCache.get(id) : null;
    if (!cached) {
      if (!bootstrapSecret) {
        setMode("setup", "error");
        setStatus("Enter the bootstrap secret to attach to sessions.");
        setSessionLoading(false);
        return;
      }
      const res = await fetchWithTimeout(
        apiURL(`/v1/sessions/${id}/attach`),
        {
          method: "POST",
          headers: bootstrapHeaders(),
        },
        LOAD_ATTACH_TIMEOUT_MS
      );
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `attach failed: ${res.status}`);
      }
      const body = await res.json();
      cached = {
        token: body.token,
        cols: body.cols,
        rows: body.rows,
      };
      sessionCache.set(id, cached);
    }

    sessionId = id;
    token = cached.token;
    acknowledgeSession(id);
    syncPtyDimensionsFromList(id);
    if (cached.cols) {
      ptyCols = cached.cols;
    }
    if (cached.rows) {
      ptyRows = cached.rows;
    }
    updateURL();

    controlling = false;
    disconnectWS();
    connectWS();
    updateSessionListActive();
  } catch (err) {
    terminalWrap.classList.remove("switching");
    if (/not found/i.test(err.message)) {
      setMode("error", "error");
      setStatus(`Attach failed: ${err.message}`);
      awaitingInitialSync = false;
      setSessionLoading(false);
      sessionId = null;
      token = null;
      updateURL();
      showPlaceholder(true);
      setMode("waiting");
      setStatus("Session ended — select one from the list or wait for a new session.");
      await refreshSessions({ autoAttach: true });
    } else {
      const msg =
        err.name === "AbortError"
          ? "Attach timed out — server did not respond in time."
          : `Attach failed: ${err.message}`;
      failSessionLoad(msg, { retry: true });
    }
  } finally {
    attaching = false;
    updateSessionLoadingState();
  }
}

async function refreshSessions({ autoAttach = true } = {}) {
  if (!bootstrapSecret) {
    syncSessions([]);
    if (!sessionId || !token) {
      showPlaceholder(true);
      setMode("setup");
      setStatus("Enter the bootstrap secret printed by tuile serve.");
    }
    return false;
  }

  try {
    const res = await fetch(apiURL("/v1/sessions"), { headers: bootstrapHeaders() });
    if (!res.ok) {
      if (res.status === 401) {
        setMode("setup", "error");
        setStatus("Bootstrap secret rejected — update it and save.");
      }
      return false;
    }
    const body = await res.json();
    syncSessions(body.sessions || []);

    const stillActive = knownSessions.some((s) => s.session_id === sessionId);
    if (sessionId && stillActive) {
      syncPtyDimensionsFromList(sessionId);
      if (!controlling) {
        scheduleTerminalLayout();
      }
    }
    if (sessionId && !stillActive) {
      disconnectWS();
      token = null;
      updateURL();
      showPlaceholder(true);
      setMode("waiting", "error");
      setStatus("Session ended. Pick another or wait for a new one.");
      sessionId = null;
    }

    if (autoAttach && !sessionId && knownSessions.length === 1) {
      await attachToSession(knownSessions[0].session_id);
      return true;
    }

    if (!sessionId && knownSessions.length === 0) {
      showPlaceholder(true);
      setMode("waiting");
      setStatus("No active sessions — this page will attach when one appears.");
    } else if (!sessionId && knownSessions.length > 1) {
      showPlaceholder(true);
      setMode("idle");
      setStatus("Multiple sessions available — select one to tail.");
    }
    return true;
  } catch (err) {
    setStatus(`Session discovery failed: ${err.message}`);
    return false;
  }
}

let refreshButtonTimer = null;
let refreshButtonFeedback = null;

function clearRefreshButtonTimers() {
  if (refreshButtonTimer) {
    clearTimeout(refreshButtonTimer);
    refreshButtonTimer = null;
  }
}

function setRefreshButtonVisual(state) {
  refreshSessionsBtn.classList.remove("is-refreshing", "is-refresh-ok", "is-refresh-error");
  if (state) {
    refreshSessionsBtn.classList.add(state);
  }
  const busy = state === "is-refreshing";
  refreshSessionsBtn.disabled = busy;
  refreshSessionsBtn.setAttribute("aria-busy", String(busy));
}

function finishRefreshButtonFeedback(result) {
  if (!refreshButtonFeedback?.active) {
    return;
  }
  refreshButtonFeedback.active = false;
  clearRefreshButtonTimers();

  if (result === "timeout") {
    setRefreshButtonVisual("is-refresh-error");
    refreshButtonTimer = setTimeout(() => setRefreshButtonVisual(null), 1400);
    return;
  }

  setRefreshButtonVisual(result === "ok" ? "is-refresh-ok" : "is-refresh-error");
  refreshButtonTimer = setTimeout(() => setRefreshButtonVisual(null), result === "ok" ? 700 : 1400);
}

async function handleRefreshSessionsClick() {
  if (refreshSessionsBtn.disabled) {
    return;
  }

  clearRefreshButtonTimers();
  if (refreshButtonFeedback?.active) {
    refreshButtonFeedback.active = false;
  }

  const feedback = { active: true };
  refreshButtonFeedback = feedback;
  setRefreshButtonVisual("is-refreshing");

  const started = performance.now();
  refreshButtonTimer = setTimeout(() => {
    if (feedback.active) {
      finishRefreshButtonFeedback("timeout");
    }
  }, REFRESH_BUTTON_TIMEOUT_MS);

  const ok = await refreshSessions({ autoAttach: false });
  if (!feedback.active) {
    return;
  }

  clearRefreshButtonTimers();

  const remaining = REFRESH_SPIN_MIN_MS - (performance.now() - started);
  if (remaining > 0) {
    await new Promise((resolve) => {
      refreshButtonTimer = setTimeout(resolve, remaining);
    });
  }
  if (!feedback.active) {
    return;
  }

  finishRefreshButtonFeedback(ok ? "ok" : "error");
}

function startPolling() {
  clearInterval(pollTimer);
  pollTimer = setInterval(() => {
    refreshSessions({ autoAttach: !sessionId });
  }, POLL_MS);
}

function sendInput(data) {
  if (!ws || ws.readyState !== WebSocket.OPEN || !controlling) {
    return;
  }
  ws.send(data);
}

term.onData((data) => {
  sendInput(data);
});

async function postJSON(path, method = "POST") {
  const res = await fetch(apiURL(path), { method, headers: authHeaders() });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `HTTP ${res.status}`);
  }
  return res.json();
}

takeoverBtn.addEventListener("click", async () => {
  try {
    await postJSON(`/v1/sessions/${sessionId}/takeover`);
    controlling = true;
    setMode("control", "control");
    applyTerminalLayout();
  } catch (err) {
    setStatus(`Takeover failed: ${err.message}`);
  }
});

releaseBtn.addEventListener("click", async () => {
  try {
    await postJSON(`/v1/sessions/${sessionId}/release`);
    controlling = false;
    setMode("observe", "observe");
    awaitingInitialSync = true;
    setSessionLoading(true, "Refreshing terminal…");
    term.reset();
    term.resize(ptyCols, ptyRows);
    await loadScreenSnapshot({ forceFinish: true });
  } catch (err) {
    setStatus(`Release failed: ${err.message}`);
    setSessionLoading(false);
    awaitingInitialSync = false;
  }
});

reconnectBtn.addEventListener("click", () => {
  if (sessionId) {
    attachToSession(sessionId, { force: true });
  } else {
    refreshSessions({ autoAttach: true });
  }
});

loadingRetry.addEventListener("click", () => {
  if (sessionId) {
    attachToSession(sessionId, { force: true });
  } else {
    refreshSessions({ autoAttach: true });
  }
});

refreshSessionsBtn.addEventListener("click", () => {
  handleRefreshSessionsClick();
});

sessionSortSelect.addEventListener("change", () => {
  saveSessionSort(sessionSortSelect.value);
  renderSessionList(knownSessions);
});

sessionInactiveMins.addEventListener("change", () => {
  const value = parseInt(sessionInactiveMins.value, 10);
  if (!Number.isFinite(value) || value < 1) {
    sessionInactiveMins.value = String(getInactiveMins());
    return;
  }
  saveInactiveMins(Math.min(value, 1440));
  sessionInactiveMins.value = String(getInactiveMins());
  renderSessionList(knownSessions);
});

bootstrapForm.addEventListener("submit", (ev) => {
  ev.preventDefault();
  bootstrapSecret = bootstrapInput.value.trim();
  if (bootstrapSecret) {
    localStorage.setItem(BOOTSTRAP_KEY, bootstrapSecret);
  } else {
    localStorage.removeItem(BOOTSTRAP_KEY);
  }
  refreshSessions({ autoAttach: true });
});

fontSelect.addEventListener("change", () => {
  term.options.fontFamily = fontSelect.value;
  updateWebGLControl();
  scheduleTerminalLayout();
});

fontSizeSelect.addEventListener("change", () => {
  fontSizeMode = fontSizeSelect.value;
  localStorage.setItem(FONT_SIZE_KEY, fontSizeMode);
  observeBaseFont =
    fontSizeMode === "auto" ? DEFAULT_FONT_SIZE : parseInt(fontSizeMode, 10) || DEFAULT_FONT_SIZE;
  scheduleTerminalLayout();
});

webglToggle.addEventListener("change", () => {
  webglToggle.setAttribute("aria-checked", String(webglToggle.checked));
  localStorage.setItem(WEBGL_KEY, webglToggle.checked ? "1" : "0");
  localStorage.removeItem(LEGACY_LIGATURES_KEY);
  setWebGLClass(webglToggle.checked);
  if (controlling) {
    setWebGLRenderer(webglToggle.checked);
    scheduleTerminalLayout();
    if (sessionId && token) {
      attachToSession(sessionId, { force: true });
    }
    return;
  }
  setWebGLRenderer(false);
  scheduleTerminalLayout();
});

function scheduleHumanResize() {
  if (!controlling) {
    return;
  }
  clearTimeout(resizeTimer);
  resizeTimer = setTimeout(async () => {
    ptyCols = term.cols;
    ptyRows = term.rows;
    try {
      await fetch(apiURL(`/v1/sessions/${sessionId}/human/resize`), {
        method: "POST",
        headers: authHeaders(),
        body: JSON.stringify({ cols: ptyCols, rows: ptyRows }),
      });
    } catch {
      // cosmetic fit still applies locally
    }
  }, 150);
}

window.addEventListener("resize", () => {
  scheduleTerminalLayout();
});

new ResizeObserver(() => {
  scheduleTerminalLayout();
}).observe(terminalWrap);

zoomOutBtn.addEventListener("click", () => {
  setObserveZoom(observeZoom - ZOOM_STEP);
});

zoomInBtn.addEventListener("click", () => {
  setObserveZoom(observeZoom + ZOOM_STEP);
});

zoomResetBtn.addEventListener("click", () => {
  setObserveZoom(1);
});

document.addEventListener("keydown", (ev) => {
  if (controlling || ev.metaKey || ev.ctrlKey || ev.altKey) {
    return;
  }
  const tag = ev.target?.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") {
    return;
  }
  if (ev.key === "-" || ev.key === "_") {
    ev.preventDefault();
    setObserveZoom(observeZoom - ZOOM_STEP);
  } else if (ev.key === "=" || ev.key === "+") {
    ev.preventDefault();
    setObserveZoom(observeZoom + ZOOM_STEP);
  } else if (ev.key === "0") {
    ev.preventDefault();
    setObserveZoom(1);
  }
});

updateZoomControl();

document.fonts?.ready?.then(() => scheduleTerminalLayout());

async function boot() {
  startPolling();
  if (sessionId && token) {
    sessionCache.set(sessionId, { token, cols: ptyCols, rows: ptyRows });
    await refreshSessions({ autoAttach: false });
    const sess = knownSessions.find((s) => s.session_id === sessionId);
    if (sess) {
      ptyCols = sess.cols || ptyCols;
      ptyRows = sess.rows || ptyRows;
    }
    showPlaceholder(false);
    connectWS();
    return;
  }
  if (sessionId && !token) {
    await attachToSession(sessionId);
    await refreshSessions({ autoAttach: false });
    return;
  }
  await refreshSessions({ autoAttach: true });
}

boot();
