# Third-party notices

Tuile bundles or depends on the following open-source components.

## Go dependencies

See `go.mod` and `go.sum` for the full dependency graph. Primary runtime libraries:

| Component | License | Use in Tuile |
|-----------|---------|--------------|
| [gitpod-io/xterm-go](https://github.com/gitpod-io/xterm-go) | MIT | Headless VT/xterm emulator for the HTTP screen API |
| [creack/pty](https://github.com/creack/pty) | MIT | POSIX PTY allocation |
| [coder/websocket](https://github.com/coder/websocket) | MIT | Browser terminal WebSocket |
| [pelletier/go-toml/v2](https://github.com/pelletier/go-toml) | MIT | `tuile.toml` configuration |
| [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) | MIT | `tuile-mcp` server |
| [chromedp/chromedp](https://github.com/chromedp/chromedp) | MIT | Integration/browser tests only |

## Browser viewer (CDN)

Loaded from jsDelivr in `web/index.html`:

| Component | License | Use in Tuile |
|-----------|---------|--------------|
| [@xterm/xterm](https://github.com/xtermjs/xterm.js) | MIT | Live terminal rendering in `/view` |
| [@xterm/addon-fit](https://github.com/xtermjs/xterm.js) | MIT | Terminal fit addon |
| [@xterm/addon-unicode11](https://github.com/xtermjs/xterm.js) | MIT | Unicode width tables |
| [@xterm/addon-webgl](https://github.com/xtermjs/xterm.js) | MIT | Optional WebGL renderer |
| [Google Fonts](https://fonts.google.com/) (Outfit) | [SIL OFL 1.1](https://scripts.sil.org/OFL) | UI typography (CDN) |

Bundled under `web/fonts/` (embedded, served at `/assets/fonts/`):

| Font | License | Use in Tuile |
|------|---------|--------------|
| JetBrainsMono Nerd Font, FiraCode Nerd Font Mono | SIL OFL 1.1 | Terminal icons and ligatures (Nerd Fonts v3.4.0) |
| JetBrains Mono, Fira Code (plain) | SIL OFL 1.1 | Non-Nerd mono fallback |

## Embedded assets

| Component | License | Use in Tuile |
|-----------|---------|--------------|
| [Lucide](https://lucide.dev) icons (inline paths in `web/icons.js`) | ISC | Viewer UI icons |
