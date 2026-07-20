package web

import "embed"

// FS contains the browser terminal viewer static assets (U6).
//
//go:embed index.html app.js session-state.js style.css icons.js state.js terminal-themes.js terminal-appearance-hint.js favicon.svg favicon.png favicon.ico fonts
var FS embed.FS
