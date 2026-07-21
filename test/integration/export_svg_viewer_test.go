//go:build integration

package integration_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"
	"github.com/newtosh/tuile/testkit"
)

type browserExportPair struct {
	Scale int    `json:"scale"`
	PNG   string `json:"png"`
	SVG   string `json:"svg"`
}

type browserExportPayload struct {
	One browserExportPair `json:"one"`
	Two browserExportPair `json:"two"`
}

var svgRasterDataRE = regexp.MustCompile(`href="data:image/(png|jpeg);base64,([^"]+)"`)

func decodeRasterImage(t *testing.T, b64 string) image.Image {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("decode raster base64: %v", err)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode raster image: %v", err)
	}
	return img
}

func extractSVGRaster(t *testing.T, svg []byte) image.Image {
	t.Helper()
	m := svgRasterDataRE.FindSubmatch(svg)
	if m == nil {
		t.Fatal("svg missing embedded raster href")
	}
	return decodeRasterImage(t, string(m[2]))
}

var svgRootDimsRE = regexp.MustCompile(`<svg[^>]*\bwidth="(\d+)"[^>]*\bheight="(\d+)"`)

func parseSVGLogicalSize(t *testing.T, svg []byte) (int, int) {
	t.Helper()
	m := svgRootDimsRE.FindSubmatch(svg)
	if m == nil {
		t.Fatal("svg missing width/height")
	}
	var w, h int
	var err error
	if w, err = strconv.Atoi(string(m[1])); err != nil {
		t.Fatalf("svg width: %v", err)
	}
	if h, err = strconv.Atoi(string(m[2])); err != nil {
		t.Fatalf("svg height: %v", err)
	}
	return w, h
}

func validateBrowserExportPair(t *testing.T, pair browserExportPair) {
	t.Helper()
	pngImg := decodeRasterImage(t, pair.PNG)
	svgBytes, err := base64.StdEncoding.DecodeString(pair.SVG)
	if err != nil {
		t.Fatalf("decode svg base64: %v", err)
	}
	if !strings.Contains(string(svgBytes), "<svg") {
		t.Fatal("svg payload missing <svg root")
	}
	if strings.Contains(string(svgBytes), `transform="scale(`) {
		t.Fatal("svg must not use nested scale transforms")
	}
	logicalW, logicalH := parseSVGLogicalSize(t, svgBytes)
	pngBounds := pngImg.Bounds()
	if pngBounds.Dx() != logicalW || pngBounds.Dy() != logicalH {
		t.Fatalf("logical size mismatch png=%v svg=%dx%d", pngBounds, logicalW, logicalH)
	}
	svgImg := extractSVGRaster(t, svgBytes)
	svgBounds := svgImg.Bounds()
	if svgBounds.Dx() < logicalW || svgBounds.Dy() < logicalH {
		t.Fatalf("embedded raster too small raster=%v logical=%dx%d", svgBounds, logicalW, logicalH)
	}
	if svgBounds.Dx()%logicalW != 0 || svgBounds.Dy()%logicalH != 0 {
		t.Fatalf("embedded raster %v must be an integer multiple of logical %dx%d", svgBounds, logicalW, logicalH)
	}
}

func TestBrowserExportSVGAlignsWithPNG(t *testing.T) {
	srv := testkit.NewServer(t)
	dir := t.TempDir()
	sess := srv.NewSession(t, dir)
	marker := "tuile-svg-export-marker"
	sess.EmitMarker(t, dir, marker)

	ctx, cancel := testkit.BrowserContext(t)
	defer cancel()

	viewURL := srv.URL + "/view"
	boot := string(srv.Boot)
	script := fmt.Sprintf(`(async () => {
		const token = %q;
		const sessionId = %q;
		const api = (path) => fetch(path, {headers: {Authorization: 'Bearer ' + token}});
		const screenRes = await api('/v1/sessions/' + sessionId + '/screen?replay=1');
		if (!screenRes.ok) throw new Error('screen fetch failed: ' + screenRes.status);
		const screenBody = await screenRes.json();
		const replayBytes = screenBody.replay_b64
			? Uint8Array.from(atob(screenBody.replay_b64), (c) => c.charCodeAt(0))
			: null;
		const { composeExportPNG, composeExportSVG } = await import('/export-compositor.js');
		const base = {
			chrome_preset: 'minimal',
			chrome_os_style: 'wireframe',
			background_mode: 'transparent',
			theme: 'dark',
			font_family: 'JetBrains Mono, ui-monospace, monospace',
			font_size_px: 14,
			title: 'tuile',
			show_grid_size: true,
		};
		const enc = async (blob) => {
			const buf = await blob.arrayBuffer();
			const bytes = new Uint8Array(buf);
			let s = '';
			for (let i = 0; i < bytes.length; i += 0x8000) {
				s += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
			}
			return btoa(s);
		};
		async function exportPair(scale) {
			const shared = { screen: screenBody.screen, replayBytes, viewerMetrics: null };
			const opts = { ...base, scale };
			const png = await composeExportPNG({ ...shared, opts: { ...opts, format: 'png' } });
			const svg = await composeExportSVG({ ...shared, opts: { ...opts, format: 'svg' } });
			return { scale, png: await enc(png), svg: await enc(svg) };
		}
		return JSON.stringify({ one: await exportPair(1), two: await exportPair(2) });
	})()`, sess.Token, sess.ID)

	var raw string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(viewURL),
		chromedp.Evaluate(fmt.Sprintf(`localStorage.setItem('tuile_bootstrap', %q)`, boot), nil),
		chromedp.Reload(),
		chromedp.WaitVisible("#session-list", chromedp.ByQuery),
		chromedp.Evaluate(script, &raw),
	); err != nil {
		t.Skipf("browser automation unavailable: %v", err)
	}

	var payload browserExportPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode browser export payload: %v\nraw=%s", err, raw)
	}

	t.Run("scale1x", func(t *testing.T) { validateBrowserExportPair(t, payload.One) })
	t.Run("scale2x", func(t *testing.T) { validateBrowserExportPair(t, payload.Two) })
}
