package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/export"
)

func (s *Server) registerExportRoutes() {
	s.mux.HandleFunc("POST /v1/sessions/{id}/export", s.handleSessionExport)
}

func (s *Server) handleSessionExport(w http.ResponseWriter, r *http.Request) {
	claims, err := s.authorizeSessionAny(r, auth.ScopeAgentRead, auth.ScopeHumanView)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	id := r.PathValue("id")
	if claims.SessionID != id {
		writeJSONError(w, http.StatusForbidden, "token not valid for this session")
		return
	}
	sess, ok := s.sess.Get(id)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}

	opts, customBG, err := parseExportRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if opts.Title == "" || opts.Title == "tuile" {
		if sess.CLIName != "" {
			opts.Title = sess.CLIName
		}
	}

	snap := sess.ScreenSnapshot(true)
	out, contentType, err := export.Render(snap, opts, customBG)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	ext := "png"
	if opts.Format == export.FormatSVG {
		ext = "svg"
	}
	filename := export.Filename(opts.Title, ext)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

func parseExportRequest(r *http.Request) (export.Options, io.Reader, error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		return parseExportMultipart(r)
	}
	var opts export.Options
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&opts); err != nil {
		return export.Options{}, nil, fmt.Errorf("invalid json body")
	}
	if opts == (export.Options{}) {
		opts = export.DefaultOptions()
	}
	if err := opts.Validate(); err != nil {
		return export.Options{}, nil, err
	}
	return opts, nil, nil
}

func parseExportMultipart(r *http.Request) (export.Options, io.Reader, error) {
	if err := r.ParseMultipartForm(export.MaxBackgroundBytes + (1 << 20)); err != nil {
		return export.Options{}, nil, fmt.Errorf("invalid multipart form")
	}
	opts := export.DefaultOptions()
	if v := r.FormValue("chrome_preset"); v != "" {
		opts.ChromePreset = v
	}
	if v := r.FormValue("chrome_os_style"); v != "" {
		opts.ChromeOSStyle = v
	}
	if v := r.FormValue("background_mode"); v != "" {
		opts.BackgroundMode = v
	}
	if v := r.FormValue("background_preset"); v != "" {
		opts.BackgroundPreset = v
	}
	if v := r.FormValue("format"); v != "" {
		opts.Format = v
	}
	if v := r.FormValue("scale"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return export.Options{}, nil, fmt.Errorf("invalid scale")
		}
		opts.Scale = n
	}
	if v := r.FormValue("font_size_px"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return export.Options{}, nil, fmt.Errorf("invalid font_size_px")
		}
		opts.FontSizePx = n
	}
	if v := r.FormValue("term_w_px"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return export.Options{}, nil, fmt.Errorf("invalid term_w_px")
		}
		opts.TermWPx = n
	}
	if v := r.FormValue("term_h_px"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return export.Options{}, nil, fmt.Errorf("invalid term_h_px")
		}
		opts.TermHPx = n
	}
	if v := r.FormValue("title"); v != "" {
		opts.Title = v
	}
	if v := r.FormValue("show_grid_size"); v != "" {
		opts.ShowGridSize = v == "1" || strings.EqualFold(v, "true")
	}
	if err := opts.Validate(); err != nil {
		return export.Options{}, nil, err
	}
	var custom io.Reader
	if opts.BackgroundMode == export.BackgroundCustom {
		file, header, err := r.FormFile("background_image")
		if err != nil {
			return export.Options{}, nil, fmt.Errorf("background_image required")
		}
		defer file.Close()
		if header.Size > export.MaxBackgroundBytes {
			return export.Options{}, nil, fmt.Errorf("background image too large")
		}
		custom = file
	}
	return opts, custom, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
