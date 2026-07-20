package export

// BackgroundSpec describes a preset backdrop.
type BackgroundSpec struct {
	Kind  string // solid | gradient
	Color string // #rrggbb for solid
	Start string // gradient start
	End   string // gradient end
	Angle int    // degrees, 135 default
}

// ThemeChromeAccent holds viewer frame border and grid badge colors for a theme.
type ThemeChromeAccent struct {
	Border      string
	FrameBg     string
	LabelText   string
	LabelBg     string
	LabelBorder string
}

// BackgroundPresets are named backgrounds shared by API and viewer.
var BackgroundPresets = map[string]BackgroundSpec{
	"slate":   {Kind: "solid", Color: "#1e293b"},
	"ink":     {Kind: "solid", Color: "#0d1117"},
	"mist":    {Kind: "solid", Color: "#e2e8f0"},
	"white":   {Kind: "solid", Color: "#ffffff"},
	"sunset":  {Kind: "gradient", Start: "#f97316", End: "#7c3aed", Angle: 135},
	"ocean":   {Kind: "gradient", Start: "#0ea5e9", End: "#1e3a8a", Angle: 135},
	"forest":  {Kind: "gradient", Start: "#22c55e", End: "#14532d", Angle: 135},
	"midnight": {Kind: "gradient", Start: "#1e1b4b", End: "#0f172a", Angle: 135},
}

// ThemeChromeAccents pairs viewer chrome accents with each export theme.
var ThemeChromeAccents = map[string]ThemeChromeAccent{
	"slate": {
		Border: "#334e60", FrameBg: "#0b0c0f",
		LabelText: "#8ec8e0", LabelBg: "rgba(22, 22, 28, 0.88)", LabelBorder: "rgba(51, 78, 96, 0.55)",
	},
	"ink": {
		Border: "#3d444d", FrameBg: "#0b0c0f",
		LabelText: "#79c0ff", LabelBg: "rgba(13, 17, 23, 0.92)", LabelBorder: "rgba(88, 166, 255, 0.45)",
	},
	"mist": {
		Border: "#64748b", FrameBg: "#0b0c0f",
		LabelText: "#cbd5e1", LabelBg: "rgba(30, 41, 59, 0.88)", LabelBorder: "rgba(100, 116, 139, 0.5)",
	},
	"white": {
		Border: "#94a3b8", FrameBg: "#0b0c0f",
		LabelText: "#e2e8f0", LabelBg: "rgba(30, 41, 59, 0.9)", LabelBorder: "rgba(148, 163, 184, 0.45)",
	},
	"sunset": {
		Border: "#ea580c", FrameBg: "#0b0c0f",
		LabelText: "#fdba74", LabelBg: "rgba(28, 15, 30, 0.9)", LabelBorder: "rgba(249, 115, 22, 0.5)",
	},
	"ocean": {
		Border: "#0284c7", FrameBg: "#0b0c0f",
		LabelText: "#7dd3fc", LabelBg: "rgba(12, 20, 40, 0.9)", LabelBorder: "rgba(14, 165, 233, 0.45)",
	},
	"forest": {
		Border: "#15803d", FrameBg: "#0b0c0f",
		LabelText: "#86efac", LabelBg: "rgba(10, 28, 18, 0.9)", LabelBorder: "rgba(34, 197, 94, 0.45)",
	},
	"midnight": {
		Border: "#4f46e5", FrameBg: "#0b0c0f",
		LabelText: "#a5b4fc", LabelBg: "rgba(15, 14, 35, 0.9)", LabelBorder: "rgba(99, 102, 241, 0.45)",
	},
}

// ThemeChromeAccentFor returns viewer chrome accents for the active export theme.
func ThemeChromeAccentFor(opts Options) ThemeChromeAccent {
	preset := "slate"
	if opts.BackgroundMode == BackgroundPreset && opts.BackgroundPreset != "" {
		preset = opts.BackgroundPreset
	}
	if accent, ok := ThemeChromeAccents[preset]; ok {
		return accent
	}
	return ThemeChromeAccents["slate"]
}

// TitleBarHeight returns chrome title bar height at 1x scale.
func TitleBarHeight(chrome string) int {
	switch chrome {
	case ChromeOSWireframe:
		return 36
	default:
		return 0
	}
}

// ChromePadding returns outer chrome inset at 1x for wireframe preset.
func ChromePadding() int {
	return 12
}

// ChromeInnerGap returns space between wireframe title bar and terminal at 1x.
func ChromeInnerGap() int {
	return 8
}

// ViewerFramePad returns inner padding inside the Tuile viewer frame at 1x.
func ViewerFramePad() int {
	return 14
}

// ViewerFrameRadius returns corner radius for the viewer frame at 1x.
func ViewerFrameRadius() int {
	return 10
}

// GridLabelFontPx returns the grid badge font size at 1x (0.65rem @ 16px).
func GridLabelFontPx() float64 {
	return 10.4
}
