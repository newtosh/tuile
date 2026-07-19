package pty

import "bytes"

// NormalizePTYInput maps line feeds to carriage returns for interactive PTY apps.
// JSON and curl callers usually send \n; TUIs (OpenCode, Codex, etc.) expect Enter as \r.
// Set raw to pass bytes through unchanged (e.g. multi-line paste with explicit control chars).
func NormalizePTYInput(data []byte, raw bool) []byte {
	if raw || len(data) == 0 {
		return data
	}
	if bytes.IndexByte(data, '\r') >= 0 {
		return bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\r'})
	}
	out := make([]byte, 0, len(data))
	for _, b := range data {
		if b == '\n' {
			out = append(out, '\r')
		} else {
			out = append(out, b)
		}
	}
	return out
}

// SplitSubmitPTYInput separates a trailing submit keystroke from payload bytes.
// Some TUIs (e.g. Cursor Agent) treat embedded newlines in a single PTY write as soft
// line breaks; Enter must be a separate write.
func SplitSubmitPTYInput(data []byte) (payload []byte, submit bool) {
	if len(data) == 0 {
		return nil, false
	}
	if data[len(data)-1] == '\n' {
		payload = data[:len(data)-1]
		if len(payload) > 0 && payload[len(payload)-1] == '\r' {
			payload = payload[:len(payload)-1]
		}
		return payload, true
	}
	if data[len(data)-1] == '\r' {
		return data[:len(data)-1], true
	}
	return data, false
}

// PTYInput describes one or more writes to an interactive PTY.
type PTYInput struct {
	Payload []byte
	Submit  bool
}

// PreparePTYInput expands API input into payload/submit writes.
func PreparePTYInput(data []byte, raw bool, forceSubmit bool) PTYInput {
	if raw {
		return PTYInput{Payload: data}
	}
	payload, submit := SplitSubmitPTYInput(data)
	submit = submit || forceSubmit
	return PTYInput{
		Payload: NormalizePTYInput(payload, false),
		Submit:  submit,
	}
}
