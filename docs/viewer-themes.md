# Viewer fonts and themes

The Tuile browser viewer (`/view`) bundles mono fonts and exposes two independent appearance settings.

## Settings

Open the gear menu in the viewer header:

| Control | What it changes |
|---------|-----------------|
| **App appearance** | Tuile chrome only — sidebar, header, menus, session list (dark or light) |
| **Terminal theme** | xterm ANSI colors inside the terminal pane |
| **Font** | Terminal font family (Nerd Font variants recommended for statusline icons) |

App appearance and terminal theme are independent. For example, you can use **Light** app appearance with the **Dracula** terminal theme.

## Bundled fonts

Mono fonts are embedded in the binary and served from `/assets/fonts/`. They use the [SIL Open Font License 1.1](https://scripts.sil.org/OFL).

- **JetBrainsMono Nerd Font** (default) — includes Nerd Font icons for nvim statuslines and devicons
- **FiraCode Nerd Font Mono**
- **JetBrains Mono** and **Fira Code** — plain mono without icon glyphs
- **System mono** — OS fallback

Nerd Font builds are pinned in `web/fonts/VERSION`. UI text still loads **Outfit** from Google Fonts CDN.

## Terminal themes

Themes are defined in `web/terminal-themes.js` with stable ids (`family:variant`, e.g. `catppuccin:mocha`). The catalog includes **Tuile Default** and **Tuile Light** (paired with app dark/light chrome), One Dark, Dracula, Catppuccin, Gruvbox, Solarized, Tokyo Night, Rosé Pine, and GitHub variants.

The **Terminal theme** picker lists only themes that match the current **App appearance** (light themes when the app is light, dark themes when the app is dark). Switching app appearance updates the theme list and selects a matching default if the previous theme no longer applies.

When app appearance and the rendered terminal palette diverge (for example, Neovim with a dark colorscheme while the app chrome is light), the viewer may suggest switching app appearance to improve the observe frame match. This samples live cell colors from the terminal buffer; it does not read application theme files.

Choices persist in `localStorage`:

- `tuile_app_appearance` (sessionStorage + localStorage)
- `tuile_terminal_theme`
- `tuile_font_family`

Truecolor and indexed color from the PTY pass through unchanged; themes set default palette colors only.

## Export integration (future)

When terminal export merges, `export-compositor.js` should import `getTerminalTheme()` from `web/terminal-themes.js` so exports match the live viewer palette.
