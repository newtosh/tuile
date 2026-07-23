package api

import (
	"testing"
	"time"
)

func TestWSInputDedupeDropsRapidDuplicateByte(t *testing.T) {
	var d wsInputDedupe
	if d.dropDuplicateByte([]byte{'a'}) {
		t.Fatal("first byte should not drop")
	}
	if !d.dropDuplicateByte([]byte{'a'}) {
		t.Fatal("immediate duplicate should drop")
	}
	time.Sleep(25 * time.Millisecond)
	if d.dropDuplicateByte([]byte{'a'}) {
		t.Fatal("duplicate after window should not drop")
	}
}

func TestWSInputDedupeAllowsDistinctBytes(t *testing.T) {
	var d wsInputDedupe
	if d.dropDuplicateByte([]byte{'a'}) {
		t.Fatal("first byte should not drop")
	}
	if d.dropDuplicateByte([]byte{'b'}) {
		t.Fatal("different byte should not drop")
	}
}

func TestWSInputDedupeResetsOnMultiByte(t *testing.T) {
	var d wsInputDedupe
	_ = d.dropDuplicateByte([]byte{'a'})
	if d.dropDuplicateByte([]byte{'p', 'a', 's', 't', 'e'}) {
		t.Fatal("multi-byte paste should not drop")
	}
	if d.dropDuplicateByte([]byte{'a'}) {
		t.Fatal("first byte after multi-byte should not drop")
	}
	if !d.dropDuplicateByte([]byte{'a'}) {
		t.Fatal("immediate duplicate after multi-byte should drop")
	}
}
