# Viewer fonts and themes

The Tuile browser viewer (`/view`) bundles mono fonts and exposes independent appearance controls for UI chrome and terminal ANSI colors.

## Settings

Open the gear menu in the viewer header:

| Control | What it changes |
|---------|-----------------|
| **App appearance** | Tuile chrome only — sidebar, header, menus, session list (`Auto`, `Dark`, or `Light`) |
| **Terminal theme** | xterm ANSI colors inside the terminal pane |
| **Font** | Terminal font family (Nerd Font variants recommended for statusline icons) |
| **Text size** | Terminal font size or auto-fit in observe mode |
| **Enhanced rendering** | WebGL renderer for sharper Unicode and box-drawing |

**App appearance** and **Terminal theme** are independent:

- **Auto** follows the OS/browser `prefers-color-scheme` setting and updates live when the system theme changes.
- **Dark** / **Light** pin the UI chrome regardless of system preference.
- The terminal theme picker lists only themes that match the resolved app appearance (light themes when chrome is light, dark themes when chrome is dark).

For example, you can use **Light** app appearance with a dark terminal colorscheme from Neovim — the viewer may suggest switching chrome when the rendered palette clashes with the observe frame.

## Bundled fonts

Mono fonts are embedded in the binary and served from `/assets/fonts/`. They use the [SIL Open Font License 1.1](https://scripts.sil.org/OFL).

| Font | Notes |
|------|--------|
| **JetBrainsMono Nerd Font** (default) | Nerd Font icons for nvim statuslines and devicons |
| **FiraCode Nerd Font Mono** | Nerd Font + ligatures |
| **JetBrains Mono**, **Fira Code** | Plain mono without icon glyphs |
| **System mono** | OS fallback |

Nerd Font builds are pinned in `web/fonts/VERSION`. UI text still loads **Outfit** from Google Fonts CDN.

## Terminal themes

Themes live in `web/terminal-themes.js` with stable ids (`family:variant`, e.g. `catppuccin:mocha`).

| Family | Variants |
|--------|----------|
| **Tuile** | Default (dark), Light |
| **One Dark** | Dark |
| **Dracula** | Dark |
| **Catppuccin** | Latte, Frappé, Macchiato, Mocha |
| **Gruvbox** | Dark, Light |
| **Solarized** | Dark, Light |
| **Tokyo Night** | Dark |
| **Rosé Pine** | Dawn, Rose, Moon |
| **GitHub** | Dark, Light |

Defaults: **Tuile Default** when app appearance resolves to dark, **Tuile Light** when it resolves to light. Switching app appearance updates the theme list and picks a matching default if the previous theme no longer applies.

### Appearance mismatch hint

When resolved app chrome and the rendered terminal palette diverge (for example, Neovim with a dark colorscheme while the UI is light), the viewer samples live cell colors from the xterm buffer and may show a toast suggesting a better app appearance match. This does not read application theme files.

## Persistence

Choices are stored in the browser:

| Key | Storage | Values |
|-----|---------|--------|
| `tuile_app_appearance` | `sessionStorage` + `localStorage` | `auto`, `dark`, `light` |
| `tuile_terminal_theme` | `localStorage` | theme id (e.g. `tuile:light`) |
| `tuile_font_family` | `localStorage` | CSS `font-family` value |

Truecolor and indexed color from the PTY pass through unchanged; themes set default palette colors only.

## Local validation

Recreate demo sessions while working on viewer appearance:

```bash
./scripts/viewer-demo-sessions.sh
```

Requires a running `tuile serve`, `tuile.toml` with `bootstrap_secret`, `curl`, `jq`, and `python3`. The script creates:

1. **Theme demo** — ANSI palette sample; launches `nvim` on `/tmp/tuile-export-readme.md` when available.
2. **Plain shell** — interactive session using the server's `$SHELL` (no bash override).

Override `TUILE_CONFIG` or `TUILE_BASE` to point at another config or listen address.

## Export integration (future)

When terminal export merges, `export-compositor.js` should import `getTerminalTheme()` from `web/terminal-themes.js` so exports match the live viewer palette.
