// Remote PTY input: document capture; xterm is display-only in control mode.

export function isModifierKey(event) {
  return (
    event.key === "Shift" || event.key === "Control" || event.key === "Alt" || event.key === "Meta"
  );
}

export function isPrintableKey(event) {
  return event.key.length === 1 && !event.ctrlKey && !event.altKey && !event.metaKey;
}

export function encodeControlKey(event) {
  if (event.metaKey) {
    return null;
  }

  if (event.ctrlKey) {
    if (event.key.length === 1) {
      const lower = event.key.toLowerCase();
      if (lower >= "a" && lower <= "z") {
        return String.fromCharCode(lower.charCodeAt(0) - 96);
      }
    }
    return null;
  }

  if (event.altKey) {
    if (event.key.length === 1) {
      return `\x1b${event.key}`;
    }
    return null;
  }

  if (isPrintableKey(event)) {
    return event.key;
  }

  switch (event.key) {
    case "Enter":
      return "\r";
    case "Backspace":
      return "\x7f";
    case "Tab":
      return event.shiftKey ? "\x1b[Z" : "\t";
    case "Escape":
      return "\x1b";
    case "ArrowUp":
      return "\x1b[A";
    case "ArrowDown":
      return "\x1b[B";
    case "ArrowRight":
      return "\x1b[C";
    case "ArrowLeft":
      return "\x1b[D";
    case "Home":
      return "\x1b[H";
    case "End":
      return "\x1b[F";
    case "Delete":
      return "\x1b[3~";
    default:
      return null;
  }
}

export const encodeTerminalKey = encodeControlKey;

function isFormField(target) {
  const tag = target?.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
}

function textareaFor(terminalWrap) {
  return terminalWrap.querySelector(".xterm-helper-textarea");
}

let detachedTextarea = null;
let detachedParent = null;
let detachedNext = null;

export function suppressXtermKeyboard(term, terminalWrap) {
  term.options.disableStdin = true;

  const live = textareaFor(terminalWrap);
  if (live) {
    live.readOnly = true;
    live.value = "";
    detachedParent = live.parentNode;
    detachedNext = live.nextSibling;
    detachedTextarea = live;
    live.remove();
  }

  if (!terminalWrap.hasAttribute("tabindex")) {
    terminalWrap.dataset.tuileTabIndex = "1";
    terminalWrap.tabIndex = 0;
  }
  if (document.activeElement?.classList?.contains("xterm-helper-textarea")) {
    document.activeElement.blur();
  }
  if (document.activeElement !== terminalWrap) {
    terminalWrap.focus({ preventScroll: true });
  }
}

export function restoreXtermKeyboard(terminalWrap) {
  if (detachedTextarea && detachedParent) {
    detachedParent.insertBefore(detachedTextarea, detachedNext);
    detachedTextarea.readOnly = false;
    detachedTextarea.value = "";
    detachedTextarea = null;
    detachedParent = null;
    detachedNext = null;
  } else {
    const ta = textareaFor(terminalWrap);
    if (ta) {
      ta.readOnly = false;
    }
  }
  if (terminalWrap.dataset.tuileTabIndex) {
    terminalWrap.removeAttribute("tabindex");
    delete terminalWrap.dataset.tuileTabIndex;
  }
}

export function installControlInput(term, terminalWrap, { isControlling, send, shouldSkipKey }) {
  const handledKeydown = new WeakSet();

  const shouldCapture = (target) => {
    if (!isControlling()) {
      return false;
    }
    if (isFormField(target) && !terminalWrap.contains(target)) {
      return false;
    }
    return true;
  };

  const blockXtermKeyEvent = (ev) => {
    if (!shouldCapture(ev.target)) {
      return;
    }
    if (shouldSkipKey(ev)) {
      return;
    }
    ev.preventDefault();
    ev.stopPropagation();
    ev.stopImmediatePropagation();
    suppressXtermKeyboard(term, terminalWrap);
  };

  const onDocumentKeyDown = (ev) => {
    if (!shouldCapture(ev.target)) {
      return;
    }
    if (handledKeydown.has(ev)) {
      ev.preventDefault();
      return;
    }
    handledKeydown.add(ev);
    if (shouldSkipKey(ev)) {
      return;
    }
    const encoded = encodeControlKey(ev);
    if (encoded === null) {
      return;
    }
    ev.preventDefault();
    ev.stopPropagation();
    ev.stopImmediatePropagation();
    suppressXtermKeyboard(term, terminalWrap);
    send(encoded);
  };

  const onDocumentPaste = (ev) => {
    if (!shouldCapture(ev.target)) {
      return;
    }
    const text = ev.clipboardData?.getData("text/plain");
    if (!text) {
      return;
    }
    ev.preventDefault();
    ev.stopPropagation();
    ev.stopImmediatePropagation();
    suppressXtermKeyboard(term, terminalWrap);
    send(text);
  };

  const onFocusIn = (ev) => {
    if (!isControlling()) {
      return;
    }
    if (ev.target?.classList?.contains("xterm-helper-textarea")) {
      ev.preventDefault();
      ev.target.blur();
      suppressXtermKeyboard(term, terminalWrap);
    }
  };

  const blockTextareaInput = (ev) => {
    if (!isControlling()) {
      return;
    }
    if (!terminalWrap.contains(ev.target)) {
      return;
    }
    ev.preventDefault();
    ev.stopImmediatePropagation();
    suppressXtermKeyboard(term, terminalWrap);
  };

  document.addEventListener("keydown", onDocumentKeyDown, true);
  document.addEventListener("keyup", blockXtermKeyEvent, true);
  document.addEventListener("keypress", blockXtermKeyEvent, true);
  document.addEventListener("paste", onDocumentPaste, true);
  terminalWrap.addEventListener("focusin", onFocusIn, true);
  terminalWrap.addEventListener("input", blockTextareaInput, true);
  terminalWrap.addEventListener("beforeinput", blockTextareaInput, true);

  term.attachCustomKeyEventHandler(() => !isControlling());

  const focusRelay = () => {
    if (!isControlling()) {
      return;
    }
    suppressXtermKeyboard(term, terminalWrap);
  };

  const dispose = () => {
    document.removeEventListener("keydown", onDocumentKeyDown, true);
    document.removeEventListener("keyup", blockXtermKeyEvent, true);
    document.removeEventListener("keypress", blockXtermKeyEvent, true);
    document.removeEventListener("paste", onDocumentPaste, true);
    terminalWrap.removeEventListener("focusin", onFocusIn, true);
    terminalWrap.removeEventListener("input", blockTextareaInput, true);
    terminalWrap.removeEventListener("beforeinput", blockTextareaInput, true);
    restoreXtermKeyboard(terminalWrap);
  };

  return { dispose, focusRelay };
}
