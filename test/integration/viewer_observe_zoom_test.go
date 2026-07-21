//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/newtosh/tuile/testkit"
)

type observeZoomMetrics struct {
	ObserveMode bool    `json:"observeMode"`
	Zoom        string  `json:"zoom"`
	Status      string  `json:"status"`
	ScreenW     float64 `json:"screenW"`
	ScreenH     float64 `json:"screenH"`
	CanvasW     float64 `json:"canvasW"`
	CanvasH     float64 `json:"canvasH"`
	VisualW     float64 `json:"visualW"`
	VisualH     float64 `json:"visualH"`
}

const observeZoomMetricsJS = `(() => {
	const wrap = document.getElementById('terminal-wrap');
	const term = wrap?.querySelector('.xterm');
	const screen = term?.querySelector('.xterm-screen');
	const canvas = term?.querySelector('canvas');
	const rect = term?.getBoundingClientRect();
	return {
		observeMode: wrap?.classList.contains('observe-mode') ?? false,
		zoom: document.getElementById('zoom-reset')?.textContent?.trim() ?? '',
		status: document.getElementById('status-message')?.textContent?.trim() ?? '',
		screenW: screen?.offsetWidth ?? 0,
		screenH: screen?.offsetHeight ?? 0,
		canvasW: canvas?.offsetWidth ?? 0,
		canvasH: canvas?.offsetHeight ?? 0,
		visualW: rect?.width ?? 0,
		visualH: rect?.height ?? 0,
	};
})()`

func readObserveZoomMetrics(t *testing.T, ctx context.Context) observeZoomMetrics {
	t.Helper()
	var m observeZoomMetrics
	if err := chromedp.Run(ctx, chromedp.Evaluate(observeZoomMetricsJS, &m)); err != nil {
		t.Fatalf("read observe zoom metrics: %v", err)
	}
	return m
}

func terminalPixelArea(m observeZoomMetrics) float64 {
	w := m.VisualW
	h := m.VisualH
	if w <= 0 || h <= 0 {
		w = m.CanvasW
		h = m.CanvasH
	}
	if w <= 0 || h <= 0 {
		w = m.ScreenW
		h = m.ScreenH
	}
	return w * h
}

func TestViewerObserveZoomChangesTerminalSize(t *testing.T) {
	srv := testkit.NewServer(t)
	dir := t.TempDir()
	sess := srv.NewSession(t, dir)
	marker := "tuile-observe-zoom-marker"
	sess.EmitMarker(t, dir, marker)

	ctx, cancel := testkit.BrowserContext(t)
	defer cancel()

	setup := fmt.Sprintf(`(async () => {
		localStorage.setItem('tuile_bootstrap', %q);
		localStorage.setItem('tuile_zoom', '1');
		localStorage.setItem('tuile_font_size', '20');
	})()`, string(srv.Boot))

	if err := chromedp.Run(ctx,
		chromedp.Navigate(sess.ViewURL()),
		chromedp.Evaluate(setup, nil),
		chromedp.Reload(),
		chromedp.WaitVisible(".xterm-rows", chromedp.ByQuery),
	); err != nil {
		t.Skipf("browser automation unavailable: %v", err)
	}

	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		var termText string
		if err := chromedp.Run(ctx, chromedp.Text(".xterm-rows", &termText, chromedp.ByQuery)); err != nil {
			t.Skipf("browser automation unavailable: %v", err)
		}
		if strings.Contains(termText, marker) {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Let observe layout settle.
	time.Sleep(500 * time.Millisecond)

	baseline := readObserveZoomMetrics(t, ctx)
	if !baseline.ObserveMode {
		t.Fatalf("expected observe mode, status=%q", baseline.Status)
	}
	if baseline.Zoom != "100%" {
		t.Fatalf("expected 100%% zoom baseline, got %q", baseline.Zoom)
	}
	baseArea := terminalPixelArea(baseline)
	if baseArea <= 0 {
		t.Fatalf("baseline terminal has no measurable size: %+v", baseline)
	}

	var zoomInDisabled bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(`document.getElementById('zoom-in').disabled`, &zoomInDisabled)); err != nil {
		t.Fatalf("read zoom in disabled: %v", err)
	}
	if !zoomInDisabled {
		t.Fatal("zoom in should be disabled while max zoom is 100%")
	}

	for i := 0; i < 6; i++ {
		if err := chromedp.Run(ctx, chromedp.Click("#zoom-out", chromedp.ByID)); err != nil {
			t.Fatalf("click zoom out to 70%%: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	zoomedBelow := readObserveZoomMetrics(t, ctx)
	if zoomedBelow.Zoom != "70%" {
		t.Fatalf("expected 70%% zoom, got %q", zoomedBelow.Zoom)
	}
	belowArea := terminalPixelArea(zoomedBelow)
	if belowArea >= baseArea*0.75 {
		t.Fatalf("zoom below 100%% did not shrink terminal (100%%=%.0f 70%%=%.0f): %+v -> %+v",
			baseArea, belowArea, baseline, zoomedBelow)
	}

	if err := chromedp.Run(ctx, chromedp.Click("#zoom-reset", chromedp.ByID)); err != nil {
		t.Fatalf("click zoom reset: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	reset := readObserveZoomMetrics(t, ctx)
	if reset.Zoom != "100%" {
		t.Fatalf("expected zoom reset to 100%%, got %q", reset.Zoom)
	}
	resetArea := terminalPixelArea(reset)
	if resetArea < baseArea*0.95 || resetArea > baseArea*1.05 {
		t.Fatalf("zoom reset did not return near baseline size (base=%.0f reset=%.0f): %+v",
			baseArea, resetArea, reset)
	}
}
