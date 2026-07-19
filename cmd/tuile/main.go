package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/newtosh/tuile/internal/api"
	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/cli"
	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/serve"
	"github.com/newtosh/tuile/internal/session"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "session":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		if err := runSession(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "tuile: %v\n", err)
			os.Exit(1)
		}
	case "serve":
		if err := runServe(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "tuile: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "tuile: unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runSession(args []string) error {
	fs := flag.NewFlagSet("session", flag.ExitOnError)
	cliName := fs.String("cli", "", "spawn agent CLI (claude|codex|cursor-cli|copilot-cli|opencode) instead of shell")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 || rest[0] != "start" {
		return fmt.Errorf("usage: tuile session start [--cli claude|codex|cursor-cli|copilot-cli|opencode] <workspace>")
	}
	if len(rest) < 2 {
		return fmt.Errorf("workspace path required")
	}

	opts := config.DefaultSession()
	if *cliName != "" {
		var err error
		opts, err = cli.SessionForCLI(*cliName)
		if err != nil {
			return err
		}
	}

	mgr := session.NewManager()
	sess, err := mgr.Create(rest[1], opts)
	if err != nil {
		return err
	}
	defer func() { _ = mgr.Close(sess.ID) }()

	fmt.Printf("session_id=%s\nworkspace=%s\n", sess.ID, sess.Workspace)
	if *cliName != "" {
		fmt.Printf("cli=%s\n", *cliName)
	}
	fmt.Fprintf(os.Stderr, "Session running; press Ctrl+C to stop.\n")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	return nil
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:7710", "HTTP listen address")
	origins := fs.String("allowed-origins", "", "comma-separated Origin allowlist for browser WebSocket")
	tlsCert := fs.String("tls-cert", "", "TLS certificate path (optional)")
	tlsKey := fs.String("tls-key", "", "TLS private key path (optional)")
	allowRoot := fs.Bool("allow-root", false, "allow running as root (development only)")
	force := fs.Bool("force", false, "kill existing tuile serve and clear listen port before starting")
	configPath := fs.String("config", "", "path to tuile.toml (default: search cwd and parents)")
	bootstrap := fs.String("bootstrap-secret", "", "bootstrap secret for POST /v1/sessions (overrides config and env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	overrides := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		overrides[f.Name] = true
	})

	if err := config.RefuseRootIfNeeded(*allowRoot); err != nil {
		return err
	}

	cfg := config.DefaultServer()
	cfg.AllowRoot = *allowRoot
	cfg.TLSCert = *tlsCert
	cfg.TLSKey = *tlsKey

	path := *configPath
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if found, err := config.FindFile(cwd); err == nil {
			path = found
		}
	} else {
		path, _ = filepath.Abs(path)
	}
	if path != "" {
		fileCfg, err := config.LoadFile(path)
		if err != nil {
			return fmt.Errorf("load config %s: %w", path, err)
		}
		config.ApplyFile(&cfg, fileCfg)
		fmt.Fprintf(os.Stderr, "tuile: loaded config from %s\n", path)
	}

	if overrides["listen"] {
		cfg.Listen = *listen
	}
	if overrides["allowed-origins"] {
		cfg.AllowedOrigins = nil
		for _, o := range strings.Split(*origins, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				cfg.AllowedOrigins = append(cfg.AllowedOrigins, trimmed)
			}
		}
	} else if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = api.DefaultDevOrigins(cfg.Listen)
	}

	if *force {
		fmt.Fprintf(os.Stderr, "tuile: --force stopping existing listeners on %s\n", cfg.Listen)
		if err := serve.ForceTakeover(cfg.Listen); err != nil {
			return err
		}
	}

	var boot auth.BootstrapSecret
	switch {
	case overrides["bootstrap-secret"] && *bootstrap != "":
		boot = auth.BootstrapSecret(*bootstrap)
	case os.Getenv("TUILE_BOOTSTRAP_SECRET") != "":
		boot = auth.BootstrapSecret(os.Getenv("TUILE_BOOTSTRAP_SECRET"))
	case cfg.BootstrapSecret != "":
		boot = auth.BootstrapSecret(cfg.BootstrapSecret)
	default:
		secret, err := auth.NewBootstrapSecret()
		if err != nil {
			return err
		}
		boot = secret
		fmt.Fprintf(os.Stderr, "bootstrap_secret=%s\n", boot)
		fmt.Fprintf(os.Stderr, "Use Authorization: Bearer <secret> for POST /v1/sessions\n")
		fmt.Fprintf(os.Stderr, "Pin it in tuile.toml (see tuile.toml.example)\n")
	}

	srv := api.NewServer(cfg, session.NewManager(), auth.NewStore(), boot)
	fmt.Fprintf(os.Stderr, "tuile listening on %s\n", cfg.Listen)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-sig:
		return nil
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage:\n")
	fmt.Fprintf(os.Stderr, "  tuile session start [--cli claude|codex|cursor-cli|copilot-cli|opencode] <workspace>\n")
	fmt.Fprintf(os.Stderr, "  tuile serve [--force] [flags]\n")
}
