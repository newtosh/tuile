package pty

import (
	"bytes"
	"testing"
)

func TestNormalizePTYInputMapsLFToCR(t *testing.T) {
	got := NormalizePTYInput([]byte("/status\n"), false)
	want := []byte("/status\r")
	if string(got) != string(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNormalizePTYInputPreservesRaw(t *testing.T) {
	in := []byte("/status\n")
	if string(NormalizePTYInput(in, true)) != string(in) {
		t.Fatal("raw mode should not modify input")
	}
}

func TestNormalizePTYInputPreservesExistingCR(t *testing.T) {
	got := NormalizePTYInput([]byte("/status\r\n"), false)
	if string(got) != "/status\r" {
		t.Fatalf("got %q", got)
	}
}

func TestSplitSubmitPTYInput(t *testing.T) {
	payload, submit := SplitSubmitPTYInput([]byte("check-in\n"))
	if string(payload) != "check-in" || !submit {
		t.Fatalf("got payload=%q submit=%v", payload, submit)
	}

	payload, submit = SplitSubmitPTYInput([]byte("check-in\r\n"))
	if string(payload) != "check-in" || !submit {
		t.Fatalf("got payload=%q submit=%v", payload, submit)
	}

	payload, submit = SplitSubmitPTYInput([]byte("typing"))
	if string(payload) != "typing" || submit {
		t.Fatalf("got payload=%q submit=%v", payload, submit)
	}
}

func TestPreparePTYInputSplitsSubmitWrite(t *testing.T) {
	in := PreparePTYInput([]byte("check-in\n"), false, false)
	if string(in.Payload) != "check-in" || !in.Submit {
		t.Fatalf("got %+v", in)
	}

	var buf bytes.Buffer
	if err := WritePreparedInput(&buf, in, WriteOpts{Strategy: StrategyStandard}); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "check-in\r" {
		t.Fatalf("writes = %q, want %q", buf.String(), "check-in\r")
	}
}

func TestWritePreparedInputBracketedPaste(t *testing.T) {
	in := PTYInput{Payload: []byte("check-in"), Submit: true}
	var buf bytes.Buffer
	if err := WritePreparedInput(&buf, in, WriteOpts{Strategy: StrategyBracketedPaste}); err != nil {
		t.Fatal(err)
	}
	want := BracketedPasteStart + "check-in" + BracketedPasteEnd + "\r"
	if buf.String() != want {
		t.Fatalf("writes = %q, want %q", buf.String(), want)
	}
}
