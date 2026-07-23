package api

import "time"

// wsInputDedupe drops duplicate single-byte WS frames (browser keydown echo bugs).
type wsInputDedupe struct {
	last    byte
	lastAt  time.Time
	hasLast bool
}

func (d *wsInputDedupe) dropDuplicateByte(data []byte) bool {
	if len(data) != 1 {
		d.hasLast = false
		return false
	}
	now := time.Now()
	if d.hasLast && data[0] == d.last && now.Sub(d.lastAt) < 20*time.Millisecond {
		return true
	}
	d.last = data[0]
	d.lastAt = now
	d.hasLast = true
	return false
}
