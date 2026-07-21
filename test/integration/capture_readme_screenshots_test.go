//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/newtosh/tuile/testkit"
)

func TestCaptureREADMEScreenshots(t *testing.T) {
	if os.Getenv("CAPTURE_README") == "" {
		t.Skip("set CAPTURE_README=1 to capture docs/images screenshots")
	}

	srv := testkit.NewServer(t)
	sess := srv.NewSession(t, "/tmp")
	sess.WaitForShell(t)

	demoInputs := []string{
		"export PYENV_SKIP_REHASH=1\n",
		"export PS1='tuile> '\n",
		"export TERM=xterm-256color\n",
		"clear\n",
		"printf '\\n\\033[1mTuile export demo\\033[0m — polished screenshots for README and docs\\n\\n'\n",
		"printf '\\033[1mANSI palette:\\033[0m\\n'\n",
		"for i in 0 1 2 3 4 5 6 7; do printf ' \\033[3%sm█\\033[0m' \"$i\"; done; printf '\\n'\n",
		"for i in 0 1 2 3 4 5 6 7; do printf ' \\033[9%sm█\\033[0m' \"$i\"; done; printf '\\n\\n'\n",
		"printf 'Nerd icons: \\ue718 \\uf489 \\ue7c8 \\uf420\\n'\n",
		"printf 'Ligatures: => !== ===\\n\\n'\n",
		"printf '256 colors: '\n",
		"for c in 196 208 226 46 51 99 201 161; do printf '\\033[38;5;%sm█\\033[0m' \"$c\"; done; printf '\\n\\n'\n",
		"printf '\\033[2mUse Export for PNG/SVG with viewer or OS window chrome.\\033[0m\\n'\n",
	}
	for _, input := range demoInputs {
		sess.Input(t, input)
		time.Sleep(100 * time.Millisecond)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/usr/bin/chromium"),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.WindowSize(1440, 900),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	outDir := filepath.Join("..", "..", "docs", "images")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	boot := string(srv.Boot)
	setup := fmt.Sprintf(`(async () => {
		localStorage.setItem('tuile_bootstrap', %q);
		localStorage.setItem('tuile_zoom', '1');
	})()`, boot)

	shot := func(name string) chromedp.Action {
		path := filepath.Join(outDir, name)
		return chromedp.ActionFunc(func(ctx context.Context) error {
			var buf []byte
			if err := chromedp.CaptureScreenshot(&buf).Do(ctx); err != nil {
				return err
			}
			return os.WriteFile(path, buf, 0o644)
		})
	}

	waitObserve := chromedp.ActionFunc(func(ctx context.Context) error {
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			var status string
			var ready bool
			_ = chromedp.Evaluate(`document.getElementById('status-message')?.textContent?.trim() || ''`, &status).Do(ctx)
			_ = chromedp.Evaluate(`Boolean(document.querySelector('#terminal-wrap.observe-mode .xterm canvas'))`, &ready).Do(ctx)
			if ready && status != "" && !strings.Contains(status, "Waiting") && !strings.Contains(status, "Loading") && !strings.Contains(status, "Initializing") {
				return nil
			}
			time.Sleep(250 * time.Millisecond)
		}
		return fmt.Errorf("timed out waiting for observe layout")
	})

	if err := chromedp.Run(ctx,
		chromedp.Navigate(sess.ViewURL()),
		chromedp.Evaluate(setup, nil),
		chromedp.Reload(),
		waitObserve,
		chromedp.Sleep(1*time.Second),
		shot("viewer-export-demo.png"),

		chromedp.Click("#export-toggle", chromedp.ByID),
		chromedp.WaitVisible("#export-dialog", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		shot("viewer-export-dialog.png"),

		chromedp.Click("#export-close", chromedp.ByID),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Click("#zoom-reset", chromedp.ByID),
		chromedp.Sleep(600*time.Millisecond),
		shot("viewer-observe-100.png"),

		chromedp.Evaluate(`(() => { for (let i = 0; i < 6; i++) document.getElementById('zoom-out')?.click(); })()`, nil),
		chromedp.Sleep(700*time.Millisecond),
		shot("viewer-observe-zoom.png"),
	); err != nil {
		t.Fatalf("capture screenshots: %v", err)
	}

	if err := saveExportSample(sess, filepath.Join(outDir, "export-sample.png")); err != nil {
		t.Fatalf("save export sample: %v", err)
	}
}

func saveExportSample(sess *testkit.Session, path string) error {
	body := []byte(`{
		"format":"png",
		"chrome_preset":"minimal",
		"background_mode":"transparent",
		"scale":1,
		"theme":"dark",
		"show_grid_size":true,
		"title":"tuile"
	}`)
	req, err := http.NewRequest(http.MethodPost, sess.ServerURL()+"/v1/sessions/"+sess.ID+"/export", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("export status %d: %s", res.StatusCode, string(data))
	}
	return os.WriteFile(path, data, 0o644)
}
