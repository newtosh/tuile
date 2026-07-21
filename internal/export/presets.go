package export

import "image/color"

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
func TitleBarHeight(chrome string, osStyle string) int {
	if chrome == ChromeMinimal {
		return 0
	}
	if osStyle == OSStyleMacOS {
		return MacOSTitleBarHeight()
	}
	if osStyle == OSStyleWindows {
		return WindowsTitleBarHeight()
	}
	return 36
}

// MacOSTitleBarHeight returns the macOS-style title bar height at 1x.
func MacOSTitleBarHeight() int {
	return 28
}

// MacOSWindowRadius returns the macOS-style window corner radius at 1x.
func MacOSWindowRadius() int {
	return 10
}

// MacOSTrafficLightSize returns traffic light diameter at 1x.
func MacOSTrafficLightSize() int {
	return 12
}

// MacOSTrafficLightInset returns traffic light offset from window edge at 1x.
func MacOSTrafficLightInset() int {
	return 8
}

// MacOSTrafficLightGap returns spacing between traffic lights at 1x.
func MacOSTrafficLightGap() int {
	return 8
}

// MacOSTerminalInset returns text margin inside the window at 1x.
func MacOSTerminalInset() int {
	return 8
}

// MacOSWindowBg returns the unified window/title bar fill for export chrome.
func MacOSWindowBg(opts Options) color.RGBA {
	if opts.Theme == "light" {
		return color.RGBA{255, 255, 255, 255}
	}
	return color.RGBA{10, 10, 10, 255}
}

// WindowsTitleBarHeight returns the Windows Terminal tab row height at 1x.
// Grounded in microsoft/terminal#9093 and showTabsInTitlebar default (MS Learn).
func WindowsTitleBarHeight() int {
	return 36
}

// WindowsWindowRadius returns the Windows-style window corner radius at 1x.
func WindowsWindowRadius() int {
	return 8
}

// WindowsTerminalInset returns text margin inside the window at 1x.
// Matches macOS: same-color band so terminal text does not hug the window edge.
func WindowsTerminalInset() int {
	return 8
}

// WindowsTabRowMarginX returns the tab row inset from the left window edge at 1x.
func WindowsTabRowMarginX() int {
	return 8
}

// WindowsTabRowMarginTop returns the tab row inset from the top window edge at 1x.
func WindowsTabRowMarginTop() int {
	return 4
}

// WindowsTabTopRadius returns the active tab top corner radius at 1x.
func WindowsTabTopRadius() int {
	return 5
}

// WindowsTabPaddingX returns horizontal padding inside the active tab label at 1x.
func WindowsTabPaddingX() int {
	return 10
}

// WindowsTabWidth returns the fixed active tab width at 1x.
func WindowsTabWidth() int {
	return 168
}

// WindowsTabCloseButtonWidth returns the tab close control width at 1x.
func WindowsTabCloseButtonWidth() int {
	return 20
}

// WindowsTabIconGap returns space between tab icon and label at 1x.
func WindowsTabIconGap() int {
	return 8
}

// WindowsTabIconSize returns the tab profile icon size at 1x.
func WindowsTabIconSize() int {
	return 16
}

// WindowsNewTabButtonWidth returns the new-tab control width at 1x.
func WindowsNewTabButtonWidth() int {
	return 28
}

// WindowsTabMenuButtonWidth returns the tab menu chevron control width at 1x.
func WindowsTabMenuButtonWidth() int {
	return 28
}

// WindowsAppName returns the application name shown in the Windows tab.
func WindowsAppName() string {
	return "tuile"
}

// WindowsCaptionButtonWidth returns caption button width at 1x.
// Grounded in TerminalApp/MinMaxCloseControl.xaml (Width 40).
func WindowsCaptionButtonWidth() int {
	return 40
}

// WindowsWindowBg returns the unified window/title bar fill for Windows chrome.
func WindowsWindowBg(opts Options) color.RGBA {
	if opts.Theme == "light" {
		return color.RGBA{243, 243, 243, 255}
	}
	return color.RGBA{12, 12, 12, 255}
}

// WindowsTabRowBg returns the unfocused tab row background at 1x.
func WindowsTabRowBg(opts Options) color.RGBA {
	if opts.Theme == "light" {
		return color.RGBA{236, 236, 236, 255}
	}
	return color.RGBA{51, 51, 51, 255}
}

// WindowsTabActiveTopAccent returns the active tab top highlight at 1x.
func WindowsTabActiveTopAccent(opts Options) color.RGBA {
	if opts.Theme == "light" {
		return color.RGBA{0, 0, 0, 31}
	}
	return color.RGBA{255, 255, 255, 36}
}

// ChromePadding returns outer chrome inset at 1x for wireframe preset.
func ChromePadding() int {
	return 12
}

// ChromeInnerGap returns space between wireframe title bar and terminal at 1x.
func ChromeInnerGap() int {
	return 8
}

// CustomBackgroundScenePad returns wallpaper margin around chrome at 1x export scale.
func CustomBackgroundScenePad() int {
	return 48
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
