package api

import (
	"net/http"
	"strconv"

	"github.com/newtosh/tuile/internal/term"
)

const (
	headerTuileVersion = "X-Tuile-Version"
	headerTuileMatched = "X-Tuile-Matched"
)

// outputFormat selects agent-facing response shapes.
// plain: raw text body (smallest for LLM context)
// text: JSON with version + text only
// json/empty: full structured screen responses
func outputFormat(raw string) string {
	switch raw {
	case "plain", "text", "compact":
		return raw
	default:
		return "json"
	}
}

func writePlainText(w http.ResponseWriter, version uint64, text string, matched *bool) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set(headerTuileVersion, strconv.FormatUint(version, 10))
	if matched != nil {
		w.Header().Set(headerTuileMatched, strconv.FormatBool(*matched))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(text))
}

func screenText(snap term.ScreenSnapshot, format string, tailN int, hasTail bool, region bool, y1, y2 int) string {
	switch {
	case format == "text" || format == "plain":
		if hasTail {
			return term.JoinTailText(term.TailLines(snap, tailN))
		}
		if region {
			return term.JoinTailText(term.RegionLines(snap, y1, y2))
		}
		return term.VisibleText(snap)
	case hasTail:
		return term.JoinTailText(term.TailLines(snap, tailN))
	case region:
		return term.JoinTailText(term.RegionLines(snap, y1, y2))
	default:
		return ""
	}
}

func writeAgentTextJSON(w http.ResponseWriter, version uint64, text string, matched *bool) {
	resp := map[string]any{
		"version": version,
		"text":    text,
	}
	if matched != nil {
		resp["matched"] = *matched
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeAgentCompactJSON(w http.ResponseWriter, version uint64, text string, matched *bool) {
	resp := map[string]any{
		"v": version,
		"t": text,
	}
	if matched != nil {
		resp["m"] = *matched
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeAgentScreen(w http.ResponseWriter, format string, version uint64, text string) {
	switch format {
	case "plain":
		writePlainText(w, version, text, nil)
	case "text":
		writeAgentTextJSON(w, version, text, nil)
	case "compact":
		writeAgentCompactJSON(w, version, text, nil)
	default:
		writeJSON(w, http.StatusOK, screenResponse{Version: version, Text: text})
	}
}

func writeAgentWait(w http.ResponseWriter, format string, version uint64, text string, matched bool) {
	switch format {
	case "plain":
		writePlainText(w, version, text, &matched)
	case "text":
		writeAgentTextJSON(w, version, text, &matched)
	case "compact":
		writeAgentCompactJSON(w, version, text, &matched)
	default:
		writeJSON(w, http.StatusOK, waitResponse{
			Version: version,
			Matched: matched,
			Text:    text,
		})
	}
}

type inputResponse struct {
	Version uint64 `json:"version"`
}

func writeInputAck(w http.ResponseWriter, version uint64) {
	writeJSON(w, http.StatusOK, inputResponse{Version: version})
}

func parseWaitFormat(req waitRequest, query string) string {
	if req.Format != "" {
		return outputFormat(req.Format)
	}
	return outputFormat(query)
}

func formatFromQuery(r *http.Request) string {
	if f := r.URL.Query().Get("format"); f != "" {
		return outputFormat(f)
	}
	return "json"
}
