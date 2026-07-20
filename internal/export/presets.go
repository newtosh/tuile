package export

// BackgroundSpec describes a preset backdrop.
type BackgroundSpec struct {
	Kind   string // solid | gradient
	Color  string // #rrggbb for solid
	Start  string // gradient start
	End    string // gradient end
	Angle  int    // degrees, 135 default
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
