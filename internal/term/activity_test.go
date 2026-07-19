package term_test

import (
	"testing"

	"github.com/newtosh/tuile/internal/term"
)

func TestTailFingerprintChangesOnNewLine(t *testing.T) {
	e := term.New(40, 10)
	if _, err := e.Write([]byte("line one\n")); err != nil {
		t.Fatal(err)
	}
	fp1 := term.TailFingerprint(e.Snapshot(), 5)

	if _, err := e.Write([]byte("line two\n")); err != nil {
		t.Fatal(err)
	}
	fp2 := term.TailFingerprint(e.Snapshot(), 5)

	if fp1 == fp2 {
		t.Fatalf("fingerprint should change after new output: %q vs %q", fp1, fp2)
	}
}

func TestTailFingerprintIgnoresANSIOnly(t *testing.T) {
	e := term.New(40, 10)
	if _, err := e.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	fp1 := term.TailFingerprint(e.Snapshot(), 5)

	if _, err := e.Write([]byte("\x1b[A")); err != nil {
		t.Fatal(err)
	}
	fp2 := term.TailFingerprint(e.Snapshot(), 5)

	if fp1 != fp2 {
		t.Fatalf("ANSI-only writes should not change fingerprint: %q vs %q", fp1, fp2)
	}
}
