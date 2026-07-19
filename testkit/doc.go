// Package testkit helps Go projects write Tuile-backed integration and browser smoke tests.
//
// Start an in-process Tuile server, create sessions, assert on headless screen output,
// and optionally drive the /view browser terminal with chromedp (requires Chrome/Chromium).
//
// Example:
//
//	func TestSmoke(t *testing.T) {
//	    srv := testkit.NewServer(t)
//	    sess := srv.NewSession(t, t.TempDir())
//	    sess.Input(t, "printf hello\\n")
//	    sess.WaitContains(t, "hello")
//	}
//
// Downstream projects: add github.com/newtosh/tuile to go.mod (use replace for local dev).
// Run integration tests in CI with headless Chrome; pre-commit is not recommended for browser tests.
package testkit
