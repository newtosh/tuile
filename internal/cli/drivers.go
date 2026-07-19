package cli

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/pty"
)

// Supported agent CLIs for R12/R13 (KTD6).
const (
	Claude      = "claude"
	Codex       = "codex"
	CursorCLI   = "cursor-cli"
	CopilotCLI  = "copilot-cli"
	OpencodeCLI = "opencode"
)

// cursorCLIBinaries are tried in order when resolving cursor-cli.
var cursorCLIBinaries = []string{"cursor-agent", "agent", "cursor-cli"}

// copilotCLIBinaries are tried in order when resolving copilot-cli.
var copilotCLIBinaries = []string{"copilot", "copilot-cli", "gh-copilot"}

// ErrNotFound indicates the CLI binary is absent from PATH.
var ErrNotFound = errors.New("cli binary not found on PATH")

// Driver describes how to spawn an agent CLI in a PTY.
type Driver struct {
	Name           string
	Bin            string
	Args           []string
	InputStrategy  pty.InputStrategy
}

// Resolve locates a supported CLI binary and default interactive args.
func Resolve(name string) (Driver, error) {
	switch name {
	case Claude:
		bin, err := exec.LookPath(Claude)
		if err != nil {
			return Driver{}, fmt.Errorf("%w: %s", ErrNotFound, Claude)
		}
		return Driver{Name: Claude, Bin: bin, InputStrategy: pty.StrategyStandard}, nil
	case Codex:
		bin, err := exec.LookPath(Codex)
		if err != nil {
			return Driver{}, fmt.Errorf("%w: %s", ErrNotFound, Codex)
		}
		return Driver{Name: Codex, Bin: bin, InputStrategy: pty.StrategyStandard}, nil
	case CursorCLI:
		for _, name := range cursorCLIBinaries {
			bin, err := exec.LookPath(name)
			if err == nil {
				return Driver{
					Name:          CursorCLI,
					Bin:           bin,
					Args:          []string{"--yolo"},
					InputStrategy: pty.StrategyBracketedPaste,
				}, nil
			}
		}
		return Driver{}, fmt.Errorf("%w: cursor-agent", ErrNotFound)
	case CopilotCLI:
		for _, name := range copilotCLIBinaries {
			bin, err := exec.LookPath(name)
			if err == nil {
				return Driver{
					Name:          CopilotCLI,
					Bin:           bin,
					Args:          []string{"--yolo"},
					InputStrategy: pty.StrategyBracketedPaste,
				}, nil
			}
		}
		return Driver{}, fmt.Errorf("%w: copilot", ErrNotFound)
	case OpencodeCLI:
		bin, err := exec.LookPath(OpencodeCLI)
		if err != nil {
			return Driver{}, fmt.Errorf("%w: %s", ErrNotFound, OpencodeCLI)
		}
		return Driver{
			Name:          OpencodeCLI,
			Bin:           bin,
			Args:          []string{"--auto"},
			InputStrategy: pty.StrategyStandard,
		}, nil
	default:
		return Driver{}, fmt.Errorf("unsupported cli %q (want %q, %q, %q, %q, or %q)", name, Claude, Codex, CursorCLI, CopilotCLI, OpencodeCLI)
	}
}

// SessionOptions returns PTY session config that launches the driver.
func (d Driver) SessionOptions() config.Session {
	opts := config.DefaultSession()
	opts.Command = d.Bin
	opts.Args = append([]string(nil), d.Args...)
	opts.CLIName = d.Name
	opts.Cols = config.DefaultCols
	opts.Rows = config.DefaultRows
	return opts
}

// InputStrategy returns how API input should be delivered for a CLI session.
func InputStrategy(name string) pty.InputStrategy {
	switch name {
	case CursorCLI, CopilotCLI:
		return pty.StrategyBracketedPaste
	default:
		return pty.StrategyStandard
	}
}

// SessionForCLI resolves name and returns session options, or an error.
func SessionForCLI(name string) (config.Session, error) {
	d, err := Resolve(name)
	if err != nil {
		return config.Session{}, err
	}
	return d.SessionOptions(), nil
}
