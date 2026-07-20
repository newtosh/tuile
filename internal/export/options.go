package export

import (
	"fmt"
	"strings"
)

const (
	ChromeMinimal     = "minimal"
	ChromeOSWireframe = "os-wireframe"

	BackgroundTransparent = "transparent"
	BackgroundPreset    = "preset"
	BackgroundCustom    = "custom"

	FormatPNG = "png"
	FormatSVG = "svg"

	MaxBackgroundBytes = 2 << 20 // 2 MiB
	MaxTitleLen        = 120
)

// Options configures a terminal screenshot export.
type Options struct {
	ChromePreset     string `json:"chrome_preset"`
	BackgroundMode   string `json:"background_mode"`
	BackgroundPreset string `json:"background_preset,omitempty"`
	Scale            int    `json:"scale"`
	Format           string `json:"format"`
	FontFamily       string `json:"font_family,omitempty"`
	FontSizePx       int    `json:"font_size_px,omitempty"`
	Theme            string `json:"theme,omitempty"`
	Title            string `json:"title,omitempty"`
	ShowGridSize     bool   `json:"show_grid_size"`
}

// DefaultOptions returns export defaults aligned with the browser viewer.
func DefaultOptions() Options {
	return Options{
		ChromePreset:   ChromeMinimal,
		BackgroundMode: BackgroundPreset,
		BackgroundPreset: "slate",
		Scale:          1,
		Format:         FormatPNG,
		FontSizePx:     14,
		Theme:          "dark",
		Title:          "tuile",
		ShowGridSize:   true,
	}
}

// Validate normalizes and checks export options.
func (o *Options) Validate() error {
	if o == nil {
		return fmt.Errorf("options required")
	}
	switch o.ChromePreset {
	case ChromeMinimal, ChromeOSWireframe:
	default:
		return fmt.Errorf("invalid chrome_preset %q", o.ChromePreset)
	}
	switch o.BackgroundMode {
	case BackgroundTransparent, BackgroundPreset, BackgroundCustom:
	default:
		return fmt.Errorf("invalid background_mode %q", o.BackgroundMode)
	}
	if o.BackgroundMode == BackgroundPreset {
		if _, ok := BackgroundPresets[o.BackgroundPreset]; !ok {
			return fmt.Errorf("invalid background_preset %q", o.BackgroundPreset)
		}
	}
	if o.Scale != 1 && o.Scale != 2 {
		return fmt.Errorf("scale must be 1 or 2")
	}
	switch o.Format {
	case FormatPNG, FormatSVG:
	default:
		return fmt.Errorf("invalid format %q", o.Format)
	}
	if o.FontSizePx <= 0 {
		o.FontSizePx = 14
	}
	if o.FontSizePx > 48 {
		return fmt.Errorf("font_size_px too large")
	}
	if o.Theme == "" {
		o.Theme = "dark"
	}
	o.Title = strings.TrimSpace(o.Title)
	if len(o.Title) > MaxTitleLen {
		return fmt.Errorf("title too long")
	}
	if o.Title == "" {
		o.Title = "tuile"
	}
	return nil
}

// Filename returns a sanitized download filename from title and extension.
func Filename(title, ext string) string {
	cleaned := strings.TrimSpace(title)
	if cleaned == "" {
		cleaned = "tuile"
	}
	replacer := strings.NewReplacer(
		"<", "", ">", "", ":", "", "\"", "", "/", "", "\\", "", "|", "", "?", "", "*", "",
	)
	cleaned = replacer.Replace(cleaned)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		cleaned = "tuile"
	}
	if len(cleaned) > 120 {
		cleaned = cleaned[:120]
	}
	return cleaned + "." + ext
}

// LayoutScale returns the pixel multiplier for output dimensions.
func (o Options) LayoutScale() int {
	if o.Scale < 1 {
		return 1
	}
	return o.Scale
}
