package term

// IsTerminalResponse reports whether data is an auto-generated terminal reply
// (OSC/CSI/DCS), not interactive user input.
func IsTerminalResponse(data []byte) bool {
	if len(data) == 0 || data[0] != '\x1b' {
		return false
	}
	if len(data) < 2 {
		return false
	}
	switch data[1] {
	case ']', '[', 'P':
		return true
	default:
		return false
	}
}
