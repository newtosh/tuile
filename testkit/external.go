package testkit

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/config"
)

// DefaultBaseURL is the standard loopback address for `tuile serve`.
const DefaultBaseURL = "http://127.0.0.1:7710"

// EnsureServe returns a Server connected to baseURL (TUILE_URL or DefaultBaseURL).
// If nothing is listening, it starts `tuile serve` on that address and registers cleanup.
// Bootstrap secret: TUILE_BOOTSTRAP_SECRET, then bootstrap_secret from tuile.toml (cwd/parents).
func EnsureServe(t *testing.T) *Server {
	t.Helper()

	baseURL := strings.TrimRight(os.Getenv("TUILE_URL"), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	var startedCmd *exec.Cmd
	var startedBoot string
	if !healthOK(baseURL) {
		t.Logf("no Tuile at %s — starting tuile serve", baseURL)
		startedCmd, startedBoot = startTuileServe(t, listenAddr(baseURL))
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			if healthOK(baseURL) {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		if !healthOK(baseURL) {
			t.Fatalf("tuile serve did not become healthy at %s", baseURL)
		}
	} else {
		t.Logf("using existing Tuile at %s", baseURL)
	}

	boot := startedBoot
	if boot == "" {
		boot = resolveBootstrapSecret(t)
	}
	if boot == "" {
		t.Fatalf("Tuile at %s needs bootstrap auth: set TUILE_BOOTSTRAP_SECRET or bootstrap_secret in tuile.toml", baseURL)
	}

	srv := &Server{URL: baseURL, Boot: auth.BootstrapSecret(boot)}
	if !srv.bootstrapWorks(t) {
		t.Fatalf("bootstrap secret rejected by Tuile at %s — check TUILE_BOOTSTRAP_SECRET / tuile.toml", baseURL)
	}

	if startedCmd != nil && startedCmd.Process != nil {
		t.Cleanup(func() {
			_ = startedCmd.Process.Signal(os.Interrupt)
			done := make(chan struct{})
			go func() {
				_ = startedCmd.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				_ = startedCmd.Process.Kill()
			}
		})
	}

	return srv
}

func healthOK(baseURL string) bool {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func listenAddr(baseURL string) string {
	u := strings.TrimPrefix(baseURL, "http://")
	u = strings.TrimPrefix(u, "https://")
	if u == "" {
		return "127.0.0.1:7710"
	}
	return u
}

func resolveBootstrapSecret(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("TUILE_BOOTSTRAP_SECRET"); v != "" {
		return v
	}
	if path := os.Getenv("TUILE_CONFIG"); path != "" {
		if f, err := config.LoadFile(path); err == nil && f.BootstrapSecret != "" {
			return f.BootstrapSecret
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	candidates := []string{cwd}
	if root := findGoModuleRoot(cwd); root != "" {
		candidates = append(candidates, root)
	}

	seen := map[string]bool{}
	for _, start := range candidates {
		if f, _, err := config.LoadNearest(start); err == nil && f.BootstrapSecret != "" {
			return f.BootstrapSecret
		}
		for _, rel := range []string{
			filepath.Join("..", "tuile", "tuile.toml"),
			filepath.Join("..", "..", "tuile", "tuile.toml"),
		} {
			p, err := filepath.Abs(filepath.Join(start, rel))
			if err != nil || seen[p] {
				continue
			}
			seen[p] = true
			if f, err := config.LoadFile(p); err == nil && f.BootstrapSecret != "" {
				t.Logf("loaded Tuile bootstrap from %s", p)
				return f.BootstrapSecret
			}
		}
	}
	return ""
}

// findGoModuleRoot walks parents from start until a go.mod file exists.
func findGoModuleRoot(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

var tuileServeMu sync.Mutex

func startTuileServe(t *testing.T, listen string) (*exec.Cmd, string) {
	t.Helper()
	tuileServeMu.Lock()
	defer tuileServeMu.Unlock()

	bin, err := exec.LookPath("tuile")
	if err != nil {
		t.Fatalf("tuile not on PATH — install or run from ../tuile: %v", err)
	}

	boot := os.Getenv("TUILE_BOOTSTRAP_SECRET")
	if boot == "" {
		boot = resolveBootstrapSecret(t)
	}
	if boot == "" {
		secret, err := auth.NewBootstrapSecret()
		if err != nil {
			t.Fatal(err)
		}
		boot = string(secret)
	}

	cmd := exec.Command(bin, "serve", "--listen", listen, "--bootstrap-secret", boot)
	cmd.Env = os.Environ()

	var stderr bytes.Buffer
	cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)

	if err := cmd.Start(); err != nil {
		t.Fatalf("start tuile serve: %v", err)
	}

	t.Logf("started %s serve --listen %s (pid %d)", bin, listen, cmd.Process.Pid)
	return cmd, boot
}

func (s *Server) bootstrapWorks(t *testing.T) bool {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, s.URL+"/v1/sessions", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+string(s.Boot))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ConnectURL attaches to an already-running Tuile without starting serve.
func ConnectURL(t *testing.T, baseURL, bootstrap string) *Server {
	t.Helper()
	baseURL = strings.TrimRight(baseURL, "/")
	if bootstrap == "" {
		bootstrap = resolveBootstrapSecret(t)
	}
	srv := &Server{URL: baseURL, Boot: auth.BootstrapSecret(bootstrap)}
	if !healthOK(baseURL) {
		t.Fatalf("no Tuile health at %s", baseURL)
	}
	if !srv.bootstrapWorks(t) {
		t.Fatalf("bootstrap rejected at %s", baseURL)
	}
	return srv
}

// PortOpen reports whether something is listening on host:port.
func PortOpen(hostport string) bool {
	conn, err := net.DialTimeout("tcp", hostport, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// DefaultListenHostPort returns the host:port portion of DefaultBaseURL.
func DefaultListenHostPort() string {
	return listenAddr(DefaultBaseURL)
}

// ViewIndexURL returns the Tuile session list / viewer root.
func ViewIndexURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/"
}

// FormatViewURL builds a browser observe link for a session.
func FormatViewURL(baseURL, sessionID, token string) string {
	return fmt.Sprintf("%s/view?session=%s&token=%s", strings.TrimRight(baseURL, "/"), sessionID, token)
}
