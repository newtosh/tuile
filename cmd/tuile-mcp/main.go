package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/tuileclient"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	baseURL := os.Getenv("TUILE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:7710"
	}

	bootstrap := os.Getenv("TUILE_BOOTSTRAP_SECRET")
	if bootstrap == "" {
		if f, _, err := config.LoadNearest("."); err == nil {
			bootstrap = f.BootstrapSecret
		}
	}
	if bootstrap == "" {
		fmt.Fprintln(os.Stderr, "tuile-mcp: set TUILE_BOOTSTRAP_SECRET or bootstrap_secret in tuile.toml")
		os.Exit(1)
	}

	client := tuileclient.New(baseURL, bootstrap)

	s := server.NewMCPServer(
		"tuile",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	registerTools(s, client)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "tuile-mcp: %v\n", err)
		os.Exit(1)
	}
}

func registerTools(s *server.MCPServer, client *tuileclient.Client) {
	s.AddTool(mcp.NewTool("tuile_session_create",
		mcp.WithDescription("Create a Tuile PTY session and return session_id plus agent token."),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Absolute workspace directory")),
		mcp.WithString("cli", mcp.Description("Agent CLI: claude, codex, cursor-cli, copilot-cli, opencode; omit for shell")),
	), handleCreate(client))

	s.AddTool(mcp.NewTool("tuile_session_list",
		mcp.WithDescription("List active Tuile sessions."),
	), handleList(client))

	s.AddTool(mcp.NewTool("tuile_session_send",
		mcp.WithDescription("Send input to a session PTY."),
		mcp.WithString("session_id", mcp.Required()),
		mcp.WithString("token", mcp.Required(), mcp.Description("Session token from create")),
		mcp.WithString("input", mcp.Required()),
		mcp.WithBoolean("raw", mcp.Description("Send bytes verbatim without Enter translation")),
	), handleSend(client))

	s.AddTool(mcp.NewTool("tuile_session_read",
		mcp.WithDescription("Read compact tail text from a session screen (token-efficient)."),
		mcp.WithString("session_id", mcp.Required()),
		mcp.WithString("token", mcp.Required()),
		mcp.WithNumber("tail", mcp.Description("Number of trailing lines (default 20, max 200)")),
	), handleRead(client))

	s.AddTool(mcp.NewTool("tuile_session_wait",
		mcp.WithDescription("Block until session output contains text or version advances."),
		mcp.WithString("session_id", mcp.Required()),
		mcp.WithString("token", mcp.Required()),
		mcp.WithString("contains", mcp.Description("Wait until screen text contains this substring")),
		mcp.WithNumber("since_version", mcp.Description("Wait until screen version exceeds this value")),
		mcp.WithNumber("timeout_ms", mcp.Description("Max wait in milliseconds (default 15000, max 60000)")),
		mcp.WithNumber("tail", mcp.Description("Tail lines to return (default 20)")),
	), handleWait(client))

	s.AddTool(mcp.NewTool("tuile_session_close",
		mcp.WithDescription("Close a Tuile session."),
		mcp.WithString("session_id", mcp.Required()),
	), handleClose(client))
}

func handleCreate(client *tuileclient.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workspace, err := req.RequireString("workspace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		cli := req.GetString("cli", "")
		out, err := client.CreateSession(workspace, cli)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(out)), nil
	}
}

func handleList(client *tuileclient.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessions, err := client.ListSessions()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(map[string]any{"sessions": sessions})), nil
	}
}

func handleSend(client *tuileclient.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID, err := req.RequireString("session_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		token, err := req.RequireString("token")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		input, err := req.RequireString("input")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw := req.GetBool("raw", false)
		if err := client.SendInput(sessionID, token, input, raw); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText("ok"), nil
	}
}

func handleRead(client *tuileclient.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID, err := req.RequireString("session_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		token, err := req.RequireString("token")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tail := req.GetInt("tail", 20)
		out, err := client.ReadScreenText(sessionID, token, tuileclient.ScreenOptions{
			Format: "text",
			Tail:   tail,
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(out)), nil
	}
}

func handleWait(client *tuileclient.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID, err := req.RequireString("session_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		token, err := req.RequireString("token")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		contains := req.GetString("contains", "")
		since := uint64(req.GetInt("since_version", 0))
		timeoutMS := req.GetInt("timeout_ms", 15000)
		if timeoutMS <= 0 {
			timeoutMS = 15000
		}
		if timeoutMS > 60000 {
			timeoutMS = 60000
		}
		tail := req.GetInt("tail", 20)

		waitReq := tuileclient.WaitRequest{
			Contains:  contains,
			Since:     since,
			TimeoutMS: timeoutMS,
			Tail:      tail,
		}
		out, err := client.WaitWithTimeout(sessionID, token, waitReq, time.Duration(timeoutMS)*time.Millisecond)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(out)), nil
	}
}

func handleClose(client *tuileclient.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID, err := req.RequireString("session_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := client.CloseSession(sessionID); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText("closed"), nil
	}
}

func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":"marshal: %s"}`, err)
	}
	return string(data)
}
