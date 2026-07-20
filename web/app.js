import { initViewerIcons, mountIcon } from "./icons.js";
import { sessions as $sessions, activeSessionId as $activeSessionId, uiStatus, uiBadge } from "./state.js";
import {
  defaultTerminalThemeIdForAppearance,
  getTerminalTheme,
  listTerminalThemesForAppearance,
  resolveTerminalThemeId,
} from "./terminal-themes.js";
import {
  analyzeTerminalBuffer,
  appearanceHintCopy,
  shouldSuggestAppearanceSwitch,
} from "./terminal-appearance-hint.js";
import {
  normalizeAppAppearancePreference,
  resolveAppAppearance,
  systemAppearance,
} from "./app-appearance.js";
import {
  ACK_STORAGE_KEY,
  loadAckMap,
  mergeSessionsWithConnected,
  pruneClientSessionState,
  saveAckMap,
} from "./session-state.js";
import { defaultExportOptions, exportFilename } from "./export-options.js";
import { installLigatures } from "./ligatures.js";

initViewerIcons();
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", () => initViewerIcons(), { once: true });
}

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
const WS_LOAD_MAX_RETRIES = 4;
const REFRESH_SPIN_MIN_MS = 450;
const REFRESH_BUTTON_TIMEOUT_MS = 8000;
const sessionCache = new Map();

const badge = document.getElementById("mode-badge");
const appVersionEl = document.getElementById("app-version");
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
const appAppearanceSelect = document.getElementById("app-appearance");
const terminalThemeSelect = document.getElementById("terminal-theme");
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
const appearanceHint = document.getElementById("appearance-hint");
const appearanceHintIcon = document.getElementById("appearance-hint-icon");
const appearanceHintText = document.getElementById("appearance-hint-text");
const appearanceHintApply = document.getElementById("appearance-hint-apply");
const appearanceHintDismiss = document.getElementById("appearance-hint-dismiss");

let bootstrapSecret = localStorage.getItem(BOOTSTRAP_KEY) || "";
let sessionId = params.get("session");
let token = params.get("token");
let controlling = false;
let ws = null;
let resizeTimer = null;
let wsLoadRetries = 0;
let wsRetryTimer = null;
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
const OBSERVE_FONT_MIN = 14;
const OBSERVE_FONT_MAX = 64;
const OBSERVE_VIEW_INSET = 4;
const GRID_FRAME_PAD = 14;
const ZOOM_KEY = "tuile_zoom";
const ZOOM_MIN = 0.5;
const ZOOM_MAX = 1.5;
const ZOOM_STEP = 0.05;
const WEBGL_KEY = "tuile_webgl";
const LEGACY_LIGATURES_KEY = "tuile_ligatures";
const APP_APPEARANCE_KEY = "tuile_app_appearance";
const TERMINAL_THEME_KEY = "tuile_terminal_theme";
const FONT_FAMILY_KEY = "tuile_font_family";
const APPEARANCE_HINT_DISMISS_PREFIX = "tuile_appearance_hint_dismiss";
const DEFAULT_FONT_FAMILY = "'JetBrainsMono Nerd Font', monospace";
let observeZoom = clampZoom(parseFloat(localStorage.getItem(ZOOM_KEY)) || 1);
let fontSizeMode = localStorage.getItem(FONT_SIZE_KEY) || "20";

if (params.get("bootstrap")) {
  bootstrapSecret = params.get("bootstrap");
  localStorage.setItem(BOOTSTRAP_KEY, bootstrapSecret);
  params.delete("bootstrap");
  const next = `${window.location.pathname}${params.toString() ? `?${params}` : ""}`;
  window.history.replaceState(null, "", next);
}

bootstrapInput && (bootstrapInput.value = bootstrapSecret);
fontSizeSelect && (fontSizeSelect.value = fontSizeMode);
observeBaseFont = fontSizeMode === "auto" ? DEFAULT_FONT_SIZE : parseInt(fontSizeMode, 10) || DEFAULT_FONT_SIZE;

sessionSortSelect && (sessionSortSelect.value = loadSessionSort());
sessionInactiveMins && (sessionInactiveMins.value = String(getInactiveMins()));

function loadAppAppearancePreference() {
  const stored =
    sessionStorage.getItem(APP_APPEARANCE_KEY) ?? localStorage.getItem(APP_APPEARANCE_KEY);
  return normalizeAppAppearancePreference(stored);
}

function persistAppAppearance(preference) {
  sessionStorage.setItem(APP_APPEARANCE_KEY, preference);
  localStorage.setItem(APP_APPEARANCE_KEY, preference);
}

function applyAppAppearance(preference) {
  const resolved = resolveAppAppearance(preference);
  document.documentElement.dataset.appearance = resolved;
  if (appAppearanceSelect) {
    appAppearanceSelect.value = preference;
  }
}

function loadTerminalThemeId(appearance = currentAppAppearance()) {
  const stored = localStorage.getItem(TERMINAL_THEME_KEY);
  if (!stored) {
    return defaultTerminalThemeIdForAppearance(appearance);
  }
  return resolveTerminalThemeId(stored, appearance);
}

function currentAppAppearance() {
  return document.documentElement.dataset.appearance === "light" ? "light" : "dark";
}

function populateTerminalThemeSelect(selectedId, appearance = currentAppAppearance()) {
  if (!terminalThemeSelect) {
    return;
  }
  const groups = new Map();
  for (const entry of listTerminalThemesForAppearance(appearance)) {
    if (!groups.has(entry.family)) {
      groups.set(entry.family, []);
    }
    groups.get(entry.family).push(entry);
  }
  terminalThemeSelect.replaceChildren();
  for (const [family, entries] of [...groups.entries()].sort((a, b) =>
    a[0].localeCompare(b[0]),
  )) {
    const group = document.createElement("optgroup");
    group.label = family;
    for (const entry of entries) {
      const option = document.createElement("option");
      option.value = entry.id;
      option.textContent = entry.label;
      option.selected = entry.id === selectedId;
      group.appendChild(option);
    }
    terminalThemeSelect.appendChild(group);
  }
}

function reconcileTerminalThemeForAppearance(appearance) {
  const themeId = resolveTerminalThemeId(
    localStorage.getItem(TERMINAL_THEME_KEY) || defaultTerminalThemeIdForAppearance(appearance),
    appearance,
  );
  populateTerminalThemeSelect(themeId, appearance);
  applyTerminalTheme(themeId);
  localStorage.setItem(TERMINAL_THEME_KEY, themeId);
}

const APPEARANCE_HINT_DISMISS_DELAY_MS = 2200;
const APPEARANCE_HINT_ANIM_MS = 400;

let appearanceHintSuggestion = null;
let appearanceHintTimer = null;
let appearanceHintDebounce = null;
let appearanceHintHideTimer = null;

function appearanceHintDismissKey(suggestion = appearanceHintSuggestion) {
  if (!sessionId || !suggestion) {
    return null;
  }
  return `${APPEARANCE_HINT_DISMISS_PREFIX}:${sessionId}:${suggestion}`;
}

function clearAppearanceHintHideTimer() {
  clearTimeout(appearanceHintHideTimer);
  appearanceHintHideTimer = null;
}

function hideAppearanceHint() {
  clearAppearanceHintHideTimer();
  appearanceHintSuggestion = null;
  appearanceHint?.classList.remove("is-visible", "is-dismissing");
  if (appearanceHintApply) {
    appearanceHintApply.disabled = false;
  }
  if (appearanceHintDismiss) {
    appearanceHintDismiss.disabled = false;
  }
  appearanceHint?.classList.add("hidden");
}

function queueAppearanceHintHide() {
  clearAppearanceHintHideTimer();
  appearanceHintHideTimer = setTimeout(() => {
    appearanceHint?.classList.remove("is-visible");
    appearanceHint?.classList.add("is-dismissing");
    appearanceHintHideTimer = setTimeout(() => {
      hideAppearanceHint();
    }, APPEARANCE_HINT_ANIM_MS);
  }, APPEARANCE_HINT_DISMISS_DELAY_MS);
}

function actionAppearanceHint() {
  const dismissKey = appearanceHintDismissKey();
  if (dismissKey) {
    sessionStorage.setItem(dismissKey, "1");
  }
  if (appearanceHintApply) {
    appearanceHintApply.disabled = true;
  }
  if (appearanceHintDismiss) {
    appearanceHintDismiss.disabled = true;
  }
  queueAppearanceHintHide();
}

function showAppearanceHint(suggestion) {
  if (!appearanceHint || !appearanceHintText || !appearanceHintApply) {
    return;
  }
  const dismissKey = appearanceHintDismissKey(suggestion);
  if (dismissKey && sessionStorage.getItem(dismissKey)) {
    return;
  }
  const copy = appearanceHintCopy(suggestion);
  appearanceHintSuggestion = suggestion;
  appearanceHintText.textContent = copy.text;
  appearanceHintApply.textContent = copy.action;
  appearanceHint.classList.remove("hidden", "is-dismissing", "is-visible");
  if (appearanceHintApply) {
    appearanceHintApply.disabled = false;
  }
  if (appearanceHintDismiss) {
    appearanceHintDismiss.disabled = false;
  }
  requestAnimationFrame(() => {
    appearanceHint?.classList.add("is-visible");
  });
}

function dismissAppearanceHint() {
  actionAppearanceHint();
}

function maybeShowAppearanceHint() {
  if (!sessionId || awaitingInitialSync || sessionLoading) {
    return;
  }
  const suggestion = shouldSuggestAppearanceSwitch(
    currentAppAppearance(),
    analyzeTerminalBuffer(term),
  );
  if (!suggestion) {
    hideAppearanceHint();
    return;
  }
  showAppearanceHint(suggestion);
}

function scheduleAppearanceHintCheck() {
  clearTimeout(appearanceHintTimer);
  appearanceHintTimer = setTimeout(() => {
    maybeShowAppearanceHint();
  }, 1200);
}

function noteTerminalAppearanceChange() {
  clearTimeout(appearanceHintDebounce);
  appearanceHintDebounce = setTimeout(scheduleAppearanceHintCheck, 2000);
}

function resetAppearanceHintState() {
  clearTimeout(appearanceHintTimer);
  clearTimeout(appearanceHintDebounce);
  appearanceHintTimer = null;
  appearanceHintDebounce = null;
  hideAppearanceHint();
}

function loadFontFamily() {
  const stored = localStorage.getItem(FONT_FAMILY_KEY);
  if (!stored || !fontSelect) {
    return DEFAULT_FONT_FAMILY;
  }
  const known = [...fontSelect.options].some((option) => option.value === stored);
  return known ? stored : DEFAULT_FONT_FAMILY;
}

function syncTerminalStageAppearance(appearance) {
  if (terminalWrap) {
    terminalWrap.dataset.terminalAppearance = appearance === "light" ? "light" : "dark";
  }
}

function applyTerminalTheme(themeId) {
  const entry = getTerminalTheme(themeId);
  term.options.theme = { ...entry.xterm };
  syncTerminalStageAppearance(entry.appearance);
  if (typeof term.refresh === "function") {
    term.refresh();
  }
  if (terminalThemeSelect) {
    terminalThemeSelect.value = themeId;
  }
}

const initialPreference = loadAppAppearancePreference();
applyAppAppearance(initialPreference);
const initialThemeId = loadTerminalThemeId(currentAppAppearance());
populateTerminalThemeSelect(initialThemeId, currentAppAppearance());
const initialFontFamily = loadFontFamily();
if (fontSelect) {
  fontSelect.value = initialFontFamily;
}
syncTerminalStageAppearance(getTerminalTheme(initialThemeId).appearance);

const appearanceMedia = window.matchMedia("(prefers-color-scheme: light)");
function syncAutoAppAppearance() {
  if (loadAppAppearancePreference() !== "auto") {
    return;
  }
  const resolved = systemAppearance();
  if (document.documentElement.dataset.appearance === resolved) {
    return;
  }
  document.documentElement.dataset.appearance = resolved;
  reconcileTerminalThemeForAppearance(resolved);
  hideAppearanceHint();
  scheduleAppearanceHintCheck();
}
appearanceMedia.addEventListener("change", syncAutoAppAppearance);

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

function formatVersionLabel(raw) {
  const v = String(raw || "").trim();
  if (!v || v === "__TUILE_VERSION__") {
    return "";
  }
  if (v === "dev") {
    return "dev";
  }
  return v.startsWith("v") ? v : `v${v}`;
}

async function initAppVersion() {
  if (!appVersionEl) {
    return;
  }
  let label = formatVersionLabel(appVersionEl.textContent);
  if (!label) {
    try {
      const res = await fetch("/version");
      if (res.ok) {
        const body = await res.json();
        label = formatVersionLabel(body.version);
      }
    } catch {
      // offline or old server without /version
    }
  }
  if (label) {
    appVersionEl.textContent = label;
    appVersionEl.hidden = false;
  }
}

void initAppVersion();

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
  fontFamily: fontSelect?.value || initialFontFamily,
  letterSpacing: 0,
  scrollback: 5000,
  customGlyphs: true,
  drawBoldTextInBrightColors: true,
  minimumContrastRatio: 1,
  allowTransparency: false,
  theme: { ...getTerminalTheme(initialThemeId).xterm },
  allowProposedApi: true,
});
const fitAddon = new FitAddon.FitAddon();
const unicode11Addon = new Unicode11Addon.Unicode11Addon();
let webglAddon = null;
let canvasAddon = null;
let removeLigatures = null;

term.loadAddon(fitAddon);
term.loadAddon(unicode11Addon);
term.unicode.activeVersion = "11";
term.open(terminalWrap);
syncTerminalInputMode();

function readWebGLPref() {
  if (localStorage.getItem(WEBGL_KEY) !== null) {
    return localStorage.getItem(WEBGL_KEY) === "1";
  }
  if (localStorage.getItem(LEGACY_LIGATURES_KEY) !== null) {
    return localStorage.getItem(LEGACY_LIGATURES_KEY) === "1";
  }
  return false;
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

function refreshLigatures() {
  removeLigatures?.();
  removeLigatures = null;
  if (!webglAddon && canvasAddon) {
    removeLigatures = installLigatures(term);
    term.refresh(0, term.rows - 1);
  }
}

function setCanvasRenderer(enabled) {
  if (enabled && !canvasAddon && !webglAddon && window.CanvasAddon) {
    try {
      canvasAddon = new CanvasAddon.CanvasAddon();
      term.loadAddon(canvasAddon);
      term.refresh(0, term.rows - 1);
    } catch (err) {
      setStatus(`Canvas renderer unavailable: ${err.message}`);
    }
    return;
  }
  if (!enabled && canvasAddon) {
    canvasAddon.dispose();
    canvasAddon = null;
  }
}

function setWebGLRenderer(enabled) {
  if (enabled && !webglAddon && window.WebglAddon) {
    try {
      removeLigatures?.();
      removeLigatures = null;
      setCanvasRenderer(false);
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
      refreshLigatures();
    }
    return;
  }
  if (!enabled && webglAddon) {
    webglAddon.dispose();
    webglAddon = null;
  }
  if (!enabled) {
    setCanvasRenderer(true);
    refreshLigatures();
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

function clearWSRetryTimer() {
  if (wsRetryTimer) {
    clearTimeout(wsRetryTimer);
    wsRetryTimer = null;
  }
}

function resetWSLoadRetries() {
  wsLoadRetries = 0;
  clearWSRetryTimer();
}

function canAutoReconnect() {
  return Boolean(sessionId && bootstrapSecret && wsLoadRetries < WS_LOAD_MAX_RETRIES);
}

function scheduleWSReconnect() {
  clearWSRetryTimer();
  wsLoadRetries += 1;
  const delay = Math.min(350 * wsLoadRetries, 2000);
  const attemptMsg =
    wsLoadRetries === 1
      ? "Server restarted — reconnecting…"
      : `Reconnecting… (${wsLoadRetries}/${WS_LOAD_MAX_RETRIES})`;
  setMode("connecting");
  setStatus(attemptMsg);
  setSessionLoading(true, attemptMsg);
  loadingRetry.hidden = true;
  terminalLoading.classList.remove("is-error");
  loadingMessage.classList.remove("is-error");
  loadingSpinner.hidden = false;
  wsRetryTimer = setTimeout(() => {
    wsRetryTimer = null;
    void attachToSession(sessionId, { force: true });
  }, delay);
}

function retryOrFailSessionLoad(message) {
  if (canAutoReconnect()) {
    scheduleWSReconnect();
    return;
  }
  failSessionLoad(message, { retry: true });
}

function failSessionLoad(message, { retry = true } = {}) {
  clearWSRetryTimer();
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

async function fetchConnectedSessionMeta() {
  if (!sessionId || !token) {
    return null;
  }
  try {
    const res = await fetch(apiURL(`/v1/sessions/${sessionId}`), { headers: authHeaders() });
    if (!res.ok) {
      return null;
    }
    return await res.json();
  } catch {
    return null;
  }
}

function connectedSessionFallback() {
  if (!sessionId) {
    return null;
  }
  return {
    session_id: sessionId,
    workspace: "",
    cols: ptyCols,
    rows: ptyRows,
    controller: "agent",
  };
}

async function sessionsWithConnected(list = []) {
  if (!sessionId || !token) {
    return list;
  }
  const meta = (await fetchConnectedSessionMeta()) || connectedSessionFallback();
  return mergeSessionsWithConnected(list, meta);
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
    syncExportToggle();
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

function applyObserveGridInlineSizes(width, height) {
  const termEl = terminalElement();
  if (!termEl || !width || !height) {
    return;
  }
  const pxW = `${width}px`;
  const pxH = `${height}px`;
  termEl.style.width = pxW;
  termEl.style.height = pxH;
  for (const sel of [".xterm-viewport", ".xterm-screen", "canvas"]) {
    const el = termEl.querySelector(sel);
    if (!el) {
      continue;
    }
    el.style.width = pxW;
    el.style.height = pxH;
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

  // Canvas renderer has no DOM row layout; pin grid metrics so the terminal is visible.
  applyObserveGridInlineSizes(width, height);
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
  syncTerminalInputMode();
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
  // Observe mode uses the canvas renderer (not DOM) so ligatures and grid metrics stay aligned.
  // WebGL loses its glyph atlas after observe layout (blank canvas at <100% zoom).
  setWebGLRenderer(false);
  term.resize(ptyCols, ptyRows);
  scaleTerminalObserve();
  refreshLigatures();
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
  resetWSLoadRetries();
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
  scheduleAppearanceHintCheck();
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
  refreshLigatures();
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
        noteTerminalAppearanceChange();
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
  clearWSRetryTimer();
  resetAppearanceHintState();
  if (ws) {
    ws.onclose = null;
    ws.close();
    ws = null;
  }
  clearTimeout(syncFallbackTimer);
  syncFallbackTimer = null;
  wsWriteChain = Promise.resolve();
  controlling = false;
  syncTerminalInputMode();
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
  refreshLigatures();

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
    retryOrFailSessionLoad("Could not connect to session — check that tuile is running.");
  }, LOAD_WS_CONNECT_TIMEOUT_MS);

  ws.onopen = () => {
    resetWSLoadRetries();
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
    // onclose follows with the actionable failure path.
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
      retryOrFailSessionLoad("Connection closed before the session finished loading.");
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
    syncExportToggle();
  }
}

async function refreshSessions({ autoAttach = true } = {}) {
  if (!bootstrapSecret) {
    const list = await sessionsWithConnected([]);
    syncSessions(list);
    if (!sessionId || !token) {
      showPlaceholder(true);
      setMode("setup");
      setStatus("Enter the bootstrap secret printed by tuile serve.");
    } else if (list.length > 0) {
      setStatus("Connected — enter bootstrap secret to discover other sessions.");
    }
    return list.length > 0;
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
    syncSessions(await sessionsWithConnected(body.sessions || []));

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
    syncExportToggle();
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
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    return;
  }
  if (controlling || isTerminalResponse(data)) {
    ws.send(data);
  }
}

function isTerminalResponse(data) {
  if (!data) {
    return false;
  }
  const code = data.charCodeAt(0);
  if (code !== 0x1b) {
    return false;
  }
  const kind = data.charCodeAt(1);
  return kind === 0x5d || kind === 0x5b || kind === 0x50;
}

function syncTerminalInputMode() {
  term.options.disableStdin = !controlling;
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
    syncTerminalInputMode();
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
    syncTerminalInputMode();
    setMode("observe", "observe");
    awaitingInitialSync = true;
    setSessionLoading(true, "Refreshing terminal…");
    term.reset();
    term.resize(ptyCols, ptyRows);
    refreshLigatures();
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
  resetWSLoadRetries();
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

fontSelect?.addEventListener("change", () => {
  term.options.fontFamily = fontSelect.value;
  localStorage.setItem(FONT_FAMILY_KEY, fontSelect.value);
  refreshLigatures();
  updateWebGLControl();
  scheduleTerminalLayout();
});

appAppearanceSelect?.addEventListener("change", () => {
  const preference = appAppearanceSelect.value;
  applyAppAppearance(preference);
  persistAppAppearance(preference);
  reconcileTerminalThemeForAppearance(currentAppAppearance());
  hideAppearanceHint();
});

appearanceHintApply?.addEventListener("click", () => {
  if (!appearanceHintSuggestion) {
    return;
  }
  applyAppAppearance(appearanceHintSuggestion);
  persistAppAppearance(appearanceHintSuggestion);
  reconcileTerminalThemeForAppearance(appearanceHintSuggestion);
  actionAppearanceHint();
});

appearanceHintDismiss?.addEventListener("click", dismissAppearanceHint);

terminalThemeSelect?.addEventListener("change", () => {
  const themeId = terminalThemeSelect.value;
  applyTerminalTheme(themeId);
  localStorage.setItem(TERMINAL_THEME_KEY, themeId);
});

fontSizeSelect?.addEventListener("change", () => {
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

const exportDialog = document.getElementById("export-dialog");
const exportForm = document.getElementById("export-form");
const exportToggle = document.getElementById("export-toggle");
const exportClose = document.getElementById("export-close");
const exportFilenameInput = document.getElementById("export-filename");
const exportAppearance = document.getElementById("export-appearance");
const exportTerminalTheme = document.getElementById("export-terminal-theme");
const exportBgMode = document.getElementById("export-background-mode");
const exportCustomWrap = document.getElementById("export-custom-wrap");
const exportGridWrap = document.getElementById("export-grid-wrap");
const exportShowGrid = document.getElementById("export-show-grid");
const exportChrome = document.getElementById("export-chrome");
const exportOsStyleWrap = document.getElementById("export-os-style-wrap");
const exportOsStyle = document.getElementById("export-os-style");
const exportPreviewStage = document.getElementById("export-preview-stage");
const exportPreviewImg = document.getElementById("export-preview-img");
const exportPreviewStatus = document.getElementById("export-preview-status");
const exportBackgroundFile = document.getElementById("export-background-file");
const EXPORT_CUSTOM_FADE_MS = 240;

let exportScreenCache = null;
let exportPreviewUrl = null;
let exportPreviewTimer = null;
let exportPreviewRequest = 0;

const EXPORT_THEME_VARS = [
  "--export-frame-accent",
  "--export-frame-dim",
  "--export-frame-label-text",
  "--export-frame-label-border",
];

function clearExportThemePreview() {
  if (!terminalWrap) {
    return;
  }
  for (const name of EXPORT_THEME_VARS) {
    terminalWrap.style.removeProperty(name);
  }
}

function viewerExportDefaults() {
  const sess = knownSessions.find((s) => s.session_id === sessionId);
  const selected = Number.parseInt(fontSizeSelect?.value, 10);
  const fontSizePx =
    fontSizeMode === "auto"
      ? term?.options?.fontSize || observeBaseFont || 14
      : Number.isFinite(selected)
        ? selected
        : 14;
  const appearance = currentAppAppearance();
  const terminalThemeId =
    localStorage.getItem(TERMINAL_THEME_KEY) || defaultTerminalThemeIdForAppearance(appearance);
  return defaultExportOptions({
    fontFamily: fontSelect?.value,
    fontSizePx,
    theme: appearance,
    terminalThemeId: resolveTerminalThemeId(terminalThemeId, appearance),
    title: sess?.cli || sess?.label || "tuile",
  });
}

function populateExportTerminalThemeSelect(selectedId, appearance = exportAppearance?.value || "dark") {
  if (!exportTerminalTheme) {
    return;
  }
  const groups = new Map();
  for (const entry of listTerminalThemesForAppearance(appearance)) {
    if (!groups.has(entry.family)) {
      groups.set(entry.family, []);
    }
    groups.get(entry.family).push(entry);
  }
  exportTerminalTheme.replaceChildren();
  for (const [family, entries] of [...groups.entries()].sort((a, b) => a[0].localeCompare(b[0]))) {
    const group = document.createElement("optgroup");
    group.label = family;
    for (const entry of entries) {
      const option = document.createElement("option");
      option.value = entry.id;
      option.textContent = entry.label;
      option.selected = entry.id === selectedId;
      group.appendChild(option);
    }
    exportTerminalTheme.appendChild(group);
  }
}

function reconcileExportTerminalTheme() {
  const appearance = exportAppearance?.value === "light" ? "light" : "dark";
  const themeId = resolveTerminalThemeId(exportTerminalTheme?.value, appearance);
  populateExportTerminalThemeSelect(themeId, appearance);
  if (exportTerminalTheme) {
    exportTerminalTheme.value = themeId;
  }
  return themeId;
}

function isExportOsChrome() {
  return exportChrome?.value === "os";
}

function collectExportOptionsFromForm() {
  const appearance = exportAppearance?.value === "light" ? "light" : "dark";
  return {
    ...viewerExportDefaults(),
    chrome_preset: exportChrome?.value || "minimal",
    chrome_os_style: exportOsStyle?.value || "wireframe",
    background_mode: exportBgMode?.value || "transparent",
    scale: Number(document.getElementById("export-scale")?.value || 1),
    format: document.getElementById("export-format")?.value || "png",
    theme: appearance,
    terminal_theme_id: resolveTerminalThemeId(exportTerminalTheme?.value, appearance),
    title: exportFilenameInput?.value || "tuile",
    show_grid_size: !isExportOsChrome() && Boolean(exportShowGrid?.checked),
  };
}

function clearExportPreview() {
  if (exportPreviewUrl) {
    URL.revokeObjectURL(exportPreviewUrl);
    exportPreviewUrl = null;
  }
  if (exportPreviewImg) {
    exportPreviewImg.hidden = true;
    exportPreviewImg.removeAttribute("src");
  }
  if (exportPreviewStatus) {
    exportPreviewStatus.hidden = false;
    exportPreviewStatus.textContent = "Preview will appear here";
  }
  exportPreviewStage?.classList.remove("is-loading");
}

function setExportPreviewLoading() {
  exportPreviewStage?.classList.add("is-loading");
  if (exportPreviewStatus && !exportPreviewImg?.src) {
    exportPreviewStatus.hidden = false;
    exportPreviewStatus.textContent = "Rendering preview…";
  }
}

function setExportPreviewImage(blob) {
  const previousUrl = exportPreviewUrl;
  exportPreviewUrl = URL.createObjectURL(blob);
  if (exportPreviewImg) {
    exportPreviewImg.onload = () => {
      if (previousUrl) {
        URL.revokeObjectURL(previousUrl);
      }
      exportPreviewImg.onload = null;
      exportPreviewStage?.classList.remove("is-loading");
    };
    exportPreviewImg.src = exportPreviewUrl;
    exportPreviewImg.hidden = false;
  }
  if (exportPreviewStatus) {
    exportPreviewStatus.hidden = true;
  }
}

async function ensureExportScreenData() {
  if (exportScreenCache) {
    return exportScreenCache;
  }
  if (!sessionId || !token) {
    throw new Error("no active session");
  }
  const res = await fetchWithTimeout(
    apiURL(`/v1/sessions/${sessionId}/screen?replay=1`),
    { headers: authHeaders() },
    LOAD_SNAPSHOT_TIMEOUT_MS
  );
  if (!res.ok) {
    throw new Error(`screen fetch failed: ${res.status}`);
  }
  const body = await res.json();
  exportScreenCache = {
    screen: body.screen,
    replayBytes: body.replay_b64 ? decodeReplayB64(body.replay_b64) : null,
  };
  return exportScreenCache;
}

function viewerExportMetrics() {
  const grid = measureTerminalGrid();
  return {
    termW: grid.width,
    termH: grid.height,
    cols: ptyCols,
    rows: ptyRows,
    fontSizePx: term?.options?.fontSize,
    fontFamily: fontSelect?.value || term?.options?.fontFamily,
  };
}

function scheduleExportPreview() {
  clearTimeout(exportPreviewTimer);
  exportPreviewTimer = setTimeout(() => {
    void renderExportPreview();
  }, 120);
}

async function renderExportPreview() {
  if (!exportDialog?.open) {
    return;
  }
  const requestId = ++exportPreviewRequest;
  setExportPreviewLoading();
  try {
    const { screen, replayBytes } = await ensureExportScreenData();
    if (requestId !== exportPreviewRequest || !exportDialog?.open) {
      return;
    }
    const opts = {
      ...collectExportOptionsFromForm(),
      format: "png",
    };
    const bgFile = exportBackgroundFile?.files?.[0] || null;
    const { composeExport } = await import("./export-compositor.js");
    const blob = await composeExport({
      screen,
      replayBytes,
      opts,
      backgroundFile: bgFile,
      viewerMetrics: viewerExportMetrics(),
    });
    if (requestId !== exportPreviewRequest || !exportDialog?.open) {
      return;
    }
    setExportPreviewImage(blob);
  } catch (err) {
    if (requestId !== exportPreviewRequest || !exportDialog?.open) {
      return;
    }
    exportPreviewStage?.classList.remove("is-loading");
    if (exportPreviewStatus) {
      exportPreviewStatus.hidden = !exportPreviewImg?.src;
      exportPreviewStatus.textContent = `Preview failed: ${err.message}`;
    }
  }
}

function syncExportChromeFields() {
  const osChrome = isExportOsChrome();
  if (exportOsStyleWrap) {
    exportOsStyleWrap.hidden = !osChrome;
    exportOsStyleWrap.setAttribute("aria-hidden", String(!osChrome));
  }
  if (exportGridWrap) {
    exportGridWrap.hidden = osChrome;
    exportGridWrap.setAttribute("aria-hidden", String(osChrome));
  }
}

function syncExportBackgroundFields() {
  const mode = exportBgMode?.value || "transparent";
  const showCustom = mode === "custom";
  if (exportCustomWrap) {
    exportCustomWrap.classList.toggle("is-visible", showCustom);
    exportCustomWrap.setAttribute("aria-hidden", String(!showCustom));
    if (!showCustom) {
      window.setTimeout(() => {
        if (exportBgMode?.value !== "custom" && exportBackgroundFile) {
          exportBackgroundFile.value = "";
        }
      }, EXPORT_CUSTOM_FADE_MS);
    }
  }
  syncExportChromeFields();
}

function syncExportToggle() {
  const toggle = document.getElementById("export-toggle");
  const dialog = document.getElementById("export-dialog");
  if (!toggle) {
    return;
  }
  const canExport = Boolean(sessionId && token);
  toggle.disabled = !canExport;
  toggle.setAttribute("aria-disabled", String(!canExport));
  toggle.title = canExport
    ? "Export screenshot"
    : "Export screenshot (select a session first)";
  if (!canExport && dialog?.open) {
    closeExportDialog();
  }
}

function openExportDialog() {
  if (!exportDialog || !sessionId) {
    setStatus("Attach to a session before exporting.");
    return;
  }
  exportScreenCache = null;
  exportPreviewRequest += 1;
  const defaults = viewerExportDefaults();
  exportChrome.value = defaults.chrome_preset;
  if (exportOsStyle) {
    exportOsStyle.value = defaults.chrome_os_style || "wireframe";
  }
  exportBgMode.value = defaults.background_mode;
  document.getElementById("export-scale").value = String(defaults.scale);
  document.getElementById("export-format").value = defaults.format;
  exportFilenameInput.value = defaults.title;
  if (exportAppearance) {
    exportAppearance.value = defaults.theme;
  }
  populateExportTerminalThemeSelect(defaults.terminal_theme_id, defaults.theme);
  if (exportTerminalTheme) {
    exportTerminalTheme.value = defaults.terminal_theme_id;
  }
  if (exportShowGrid) {
    exportShowGrid.checked = defaults.show_grid_size;
  }
  if (exportBackgroundFile) {
    exportBackgroundFile.value = "";
  }
  syncExportBackgroundFields();
  syncExportChromeFields();
  exportDialog.showModal();
  exportFilenameInput?.focus({ preventScroll: true });
  scheduleExportPreview();
}

function closeExportDialog() {
  exportPreviewRequest += 1;
  clearTimeout(exportPreviewTimer);
  exportPreviewTimer = null;
  exportScreenCache = null;
  clearExportPreview();
  exportDialog?.close();
  clearExportThemePreview();
}

exportToggle?.addEventListener("click", () => openExportDialog());
exportClose?.addEventListener("click", () => closeExportDialog());
exportChrome?.addEventListener("change", () => {
  syncExportChromeFields();
  scheduleExportPreview();
});
exportOsStyle?.addEventListener("change", scheduleExportPreview);
exportBgMode?.addEventListener("change", () => {
  syncExportBackgroundFields();
  scheduleExportPreview();
});
exportAppearance?.addEventListener("change", () => {
  reconcileExportTerminalTheme();
  scheduleExportPreview();
});
exportTerminalTheme?.addEventListener("change", scheduleExportPreview);
document.getElementById("export-scale")?.addEventListener("change", scheduleExportPreview);
document.getElementById("export-format")?.addEventListener("change", scheduleExportPreview);
exportShowGrid?.addEventListener("change", scheduleExportPreview);
exportBackgroundFile?.addEventListener("change", scheduleExportPreview);

exportForm?.addEventListener("submit", async (ev) => {
  ev.preventDefault();
  if (!sessionId || !token) {
    setStatus("No active session to export.");
    return;
  }
  const opts = collectExportOptionsFromForm();
  const bgFile = exportBackgroundFile?.files?.[0] || null;
  try {
    setStatus("Exporting screenshot…");
    const { screen, replayBytes } = await ensureExportScreenData();
    const { composeExport, downloadBlob } = await import("./export-compositor.js");
    const blob = await composeExport({
      screen,
      replayBytes,
      opts,
      backgroundFile: bgFile,
      viewerMetrics: viewerExportMetrics(),
    });
    downloadBlob(blob, exportFilename(opts.title, opts.format));
    setStatus("Export downloaded.");
    closeExportDialog();
  } catch (err) {
    setStatus(`Export failed: ${err.message}`);
  }
});

document.fonts?.ready?.then(() => scheduleTerminalLayout());

async function boot() {
  startPolling();
  const targetSession = sessionId;
  if (targetSession) {
    await refreshSessions({ autoAttach: false });
    if (!sessionId) {
      await refreshSessions({ autoAttach: true });
      return;
    }
    if (bootstrapSecret) {
      await attachToSession(sessionId, { force: true });
      return;
    }
    if (token) {
      sessionCache.set(sessionId, { token, cols: ptyCols, rows: ptyRows });
      const sess = knownSessions.find((s) => s.session_id === sessionId);
      if (sess) {
        ptyCols = sess.cols || ptyCols;
        ptyRows = sess.rows || ptyRows;
      }
      showPlaceholder(false);
      connectWS();
      syncExportToggle();
      return;
    }
    await attachToSession(sessionId);
    return;
  }
  await refreshSessions({ autoAttach: true });
  syncExportToggle();
}

boot();
