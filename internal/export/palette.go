package export

import (
	"fmt"
	"image/color"
	"regexp"
	"strconv"
	"strings"
)

var rgbaRe = regexp.MustCompile(`^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)(?:\s*,\s*([\d.]+))?\s*\)$`)

// ansi16 maps classic xterm palette indices to RGB.
var ansi16 = [16]color.RGBA{
	{0, 0, 0, 255},
	{205, 49, 49, 255},
	{13, 188, 121, 255},
	{229, 229, 16, 255},
	{36, 114, 200, 255},
	{188, 63, 188, 255},
	{17, 168, 205, 255},
	{229, 229, 229, 255},
	{102, 102, 102, 255},
	{241, 76, 76, 255},
	{35, 209, 139, 255},
	{245, 245, 67, 255},
	{59, 142, 234, 255},
	{214, 112, 214, 255},
	{41, 184, 219, 255},
	{255, 255, 255, 255},
}

var defaultFG = color.RGBA{201, 209, 217, 255}
var defaultBG = color.RGBA{13, 17, 23, 255}

func parseColor(raw string, isFG bool) color.Color {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if isFG {
			return defaultFG
		}
		return defaultBG
	}
	if strings.HasPrefix(raw, "#") && len(raw) == 7 {
		var r, g, b uint8
		_, err := fmt.Sscanf(raw, "#%02x%02x%02x", &r, &g, &b)
		if err == nil {
			return color.RGBA{r, g, b, 255}
		}
	}
	if m := rgbaRe.FindStringSubmatch(raw); len(m) > 0 {
		r, _ := strconv.Atoi(m[1])
		g, _ := strconv.Atoi(m[2])
		b, _ := strconv.Atoi(m[3])
		a := 255
		if m[4] != "" {
			if af, err := strconv.ParseFloat(m[4], 64); err == nil {
				a = int(af * 255)
			}
		}
		return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
	}
	if strings.HasPrefix(raw, "p") {
		idx, err := strconv.Atoi(raw[1:])
		if err == nil && idx >= 0 && idx < 16 {
			return ansi16[idx]
		}
	}
	if isFG {
		return defaultFG
	}
	return defaultBG
}
