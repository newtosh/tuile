package web

import "embed"

// FS contains the browser terminal viewer static assets (U6).
//
//go:embed index.html app.js style.css icons.js state.js favicon.svg favicon.png favicon.ico
var FS embed.FS
