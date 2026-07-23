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

type controlInputProbe struct {
	ControlMode bool   `json:"controlMode"`
	Controlling bool   `json:"controlling"`
	TakeoverOK  bool   `json:"takeoverOK"`
	Line        string `json:"line"`
	Typed       string `json:"typed"`
	RowsTail    string `json:"rowsTail"`
}

const controlInputProbeJS = `(() => {
	const wrap = document.getElementById('terminal-wrap');
	const text = window.__tuileTest?.screenText?.() || document.querySelector('.xterm-rows')?.textContent || '';
	const lines = text.split('\n').filter((l) => l.trim().length > 0);
	const line = lines[lines.length - 1] || '';
	const gt = line.lastIndexOf('>');
	const typed = gt >= 0 ? line.slice(gt + 1).trim() : line.trim();
	const releaseBtn = document.getElementById('release');
	return {
		controlMode: wrap?.classList.contains('control-mode') ?? false,
		controlling: releaseBtn ? !releaseBtn.disabled : false,
		takeoverOK: !document.getElementById('takeover')?.disabled,
		line,
		typed,
		rowsTail: text.slice(-240),
	};
})()`

func waitControlReady(ctx context.Context) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		deadline := time.Now().Add(25 * time.Second)
		for time.Now().Before(deadline) {
			var probe controlInputProbe
			_ = chromedp.Evaluate(controlInputProbeJS, &probe).Do(ctx)
			if probe.ControlMode && probe.Controlling {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
		return fmt.Errorf("timed out waiting for control mode")
	})
}

func waitViewerConnected(ctx context.Context) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		deadline := time.Now().Add(25 * time.Second)
		for time.Now().Before(deadline) {
			var probe controlInputProbe
			_ = chromedp.Evaluate(controlInputProbeJS, &probe).Do(ctx)
			if probe.TakeoverOK {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
		return fmt.Errorf("timed out waiting for viewer websocket")
	})
}

func readControlProbe(t *testing.T, ctx context.Context) controlInputProbe {
	t.Helper()
	var probe controlInputProbe
	if err := chromedp.Run(ctx, chromedp.Evaluate(controlInputProbeJS, &probe)); err != nil {
		t.Fatalf("read control probe: %v", err)
	}
	return probe
}

func typeSlowly(ctx context.Context, text string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		for _, ch := range text {
			key := string(ch)
			script := fmt.Sprintf(`(() => {
				const key = %q;
				const ev = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true });
				document.dispatchEvent(ev);
			})()`, key)
			if err := chromedp.Evaluate(script, nil).Do(ctx); err != nil {
				return err
			}
			time.Sleep(35 * time.Millisecond)
		}
		return nil
	})
}

func waitTerminalReady(ctx context.Context, wantSubstring string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			var text string
			_ = chromedp.Evaluate(`(() => window.__tuileTest?.screenText?.() || document.querySelector('.xterm-rows')?.textContent || '')()`, &text).Do(ctx)
			if strings.TrimSpace(text) != "" {
				if wantSubstring == "" || strings.Contains(text, wantSubstring) {
					return nil
				}
			}
			time.Sleep(250 * time.Millisecond)
		}
		return fmt.Errorf("timed out waiting for terminal screen containing %q", wantSubstring)
	})
}

func readScreenText(t *testing.T, ctx context.Context) string {
	t.Helper()
	var text string
	if err := chromedp.Run(ctx, chromedp.Evaluate(`(() => window.__tuileTest?.screenText?.() || document.querySelector('.xterm-rows')?.textContent || '')()`, &text)); err != nil {
		t.Fatalf("read screen text: %v", err)
	}
	return text
}

func viewTestURL(sess *testkit.Session) string {
	return sess.ViewURL() + "&test=1"
}

func typedFromRows(text string) string {
	lines := strings.Split(text, "\n")
	seps := []string{">", "❯", "$", "#", "%"}
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimRight(lines[i], " \t")
		if line == "" {
			continue
		}
		for _, sep := range seps {
			if idx := strings.LastIndex(line, sep); idx >= 0 {
				return strings.TrimSpace(line[idx+len(sep):])
			}
		}
		return strings.TrimSpace(line)
	}
	return ""
}

func TestViewerControlInputNoDuplicateKeystrokes(t *testing.T) {
	srv := testkit.NewServer(t)
	dir := t.TempDir()
	sess := srv.NewSession(t, dir)
	sess.WaitForShell(t)

	ctx, cancel := testkit.BrowserContext(t)
	defer cancel()

	const typed = "tuile-ok"
	const readyMarker = "tuile-control-input-ready"
	sess.EmitMarker(t, dir, readyMarker)

	if err := chromedp.Run(ctx,
		chromedp.Navigate(viewTestURL(sess)),
		chromedp.WaitReady("body", chromedp.ByQuery),
		waitViewerConnected(ctx),
		waitTerminalReady(ctx, readyMarker),
		chromedp.Click("#settings-toggle", chromedp.ByID),
		chromedp.WaitVisible("#takeover", chromedp.ByID),
		chromedp.Click("#takeover", chromedp.ByID),
		waitControlReady(ctx),
		chromedp.Click("#terminal-wrap", chromedp.ByID),
		chromedp.Sleep(300*time.Millisecond),
		typeSlowly(ctx, typed),
		chromedp.Sleep(800*time.Millisecond),
	); err != nil {
		t.Skipf("browser automation unavailable: %v", err)
	}

	screenText := readScreenText(t, ctx)
	typedLine := typedFromRows(screenText)
	probe := readControlProbe(t, ctx)
	if !probe.ControlMode || !probe.Controlling {
		t.Fatalf("expected control mode, probe=%+v", probe)
	}
	if typedLine != typed {
		t.Fatalf("typed line = %q, want %q\nscreen=%q\nprobe=%+v", typedLine, typed, screenText, probe)
	}

	// Commit line and verify PTY received it once.
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }))`, nil),
		chromedp.Sleep(800*time.Millisecond),
	); err != nil {
		t.Fatalf("enter: %v", err)
	}

	screen := sess.PlainScreen(t, 20)
	if !strings.Contains(screen, typed) {
		t.Fatalf("PTY screen missing %q after enter:\n%s", typed, screen)
	}
	if strings.Contains(screen, "ttuil") || strings.Contains(screen, "ook") {
		t.Fatalf("PTY screen shows duplicated keystrokes:\n%s", screen)
	}
}

func TestViewerControlInputShortCommandXY(t *testing.T) {
	srv := testkit.NewServer(t)
	dir := t.TempDir()
	sess := srv.NewSession(t, dir)
	sess.WaitForShell(t)

	ctx, cancel := testkit.BrowserContext(t)
	defer cancel()

	const readyMarker = "tuile-control-input-ready"
	sess.EmitMarker(t, dir, readyMarker)

	if err := chromedp.Run(ctx,
		chromedp.Navigate(viewTestURL(sess)),
		chromedp.WaitReady("body", chromedp.ByQuery),
		waitViewerConnected(ctx),
		waitTerminalReady(ctx, readyMarker),
		chromedp.Click("#settings-toggle", chromedp.ByID),
		chromedp.WaitVisible("#takeover", chromedp.ByID),
		chromedp.Click("#takeover", chromedp.ByID),
		waitControlReady(ctx),
		chromedp.Click("#terminal-wrap", chromedp.ByID),
		chromedp.Sleep(300*time.Millisecond),
		typeSlowly(ctx, "xy"),
		chromedp.Sleep(800*time.Millisecond),
	); err != nil {
		t.Skipf("browser automation unavailable: %v", err)
	}

	typedLine := typedFromRows(readScreenText(t, ctx))
	if typedLine != "xy" {
		t.Fatalf("typed line = %q, want %q", typedLine, "xy")
	}
}
