package export

import (
	"bytes"
	"fmt"
	"io"

	"github.com/newtosh/tuile/internal/term"
)

// Render produces PNG or SVG bytes for the given snapshot and options.
func Render(snap term.ScreenSnapshot, opts Options, customBackground io.Reader) ([]byte, string, error) {
	if err := opts.Validate(); err != nil {
		return nil, "", err
	}
	switch opts.Format {
	case FormatSVG:
		b, err := renderSVG(snap, opts, customBackground)
		return b, "image/svg+xml", err
	case FormatPNG:
		b, err := renderPNG(snap, opts, customBackground)
		return b, "image/png", err
	default:
		return nil, "", fmt.Errorf("unsupported format %q", opts.Format)
	}
}

func renderPNG(snap term.ScreenSnapshot, opts Options, custom io.Reader) ([]byte, error) {
	if opts.BackgroundMode == BackgroundCustom && custom != nil {
		data, err := io.ReadAll(io.LimitReader(custom, MaxBackgroundBytes+1))
		if err != nil {
			return nil, err
		}
		if len(data) > MaxBackgroundBytes {
			return nil, fmt.Errorf("background image too large")
		}
		custom = bytes.NewReader(data)
	}
	return RenderPNGWithBackground(snap, opts, custom)
}

func renderSVG(snap term.ScreenSnapshot, opts Options, custom io.Reader) ([]byte, error) {
	if opts.BackgroundMode == BackgroundCustom && custom != nil {
		data, err := io.ReadAll(io.LimitReader(custom, MaxBackgroundBytes+1))
		if err != nil {
			return nil, err
		}
		if len(data) > MaxBackgroundBytes {
			return nil, fmt.Errorf("background image too large")
		}
		return RenderSVGWithBackground(snap, opts, data)
	}
	return RenderSVGWithBackground(snap, opts, nil)
}
