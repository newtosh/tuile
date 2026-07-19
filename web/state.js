import { atom } from "https://esm.sh/nanostores@1.0.1";

export const sessions = atom([]);
export const activeSessionId = atom(null);
export const uiStatus = atom("Initializing…");
export const uiBadge = atom({ text: "connecting", className: "" });
