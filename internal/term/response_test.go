package term

import "testing"

func TestIsTerminalResponse(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"osc bg", []byte("\x1b]11;rgb:0a/0a/0a\x1b\\"), true},
		{"csi cpr", []byte("\x1b[24;80R"), true},
		{"dcs", []byte("\x1bP1$r0m\x1b\\"), true},
		{"escape key", []byte("\x1b"), false},
		{"plain text", []byte("hello"), false},
		{"ctrl-c", []byte{3}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTerminalResponse(tc.data); got != tc.want {
				t.Fatalf("IsTerminalResponse(%q) = %v, want %v", tc.data, got, tc.want)
			}
		})
	}
}
