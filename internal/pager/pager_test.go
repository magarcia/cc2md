package pager

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestStart_NonTTY(t *testing.T) {
	f := openTempFile(t)

	p, w, err := Start(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		p.Stop()
		t.Fatal("expected nil Pager for non-TTY")
	}
	if w != io.Writer(f) {
		t.Error("expected writer to be the out file")
	}
}

func TestStart_NonExistentBinary(t *testing.T) {
	t.Setenv("PAGER", "this-binary-does-not-exist-cc2md-test")
	f := openTempFile(t)

	p, w, err := Start(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		p.Stop()
		t.Fatal("expected nil Pager for non-existent binary")
	}
	if w == nil {
		t.Fatal("expected non-nil writer")
	}
}

func TestStart_CatShortcut(t *testing.T) {
	t.Setenv("PAGER", "cat")
	_, w, _ := openPipe(t)

	p, writer, err := Start(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		p.Stop()
		t.Fatal("expected nil Pager for PAGER=cat")
	}
	if writer != io.Writer(w) {
		t.Error("expected direct writer")
	}
}

func TestBuildChildEnv_RemovesPAGER(t *testing.T) {
	t.Setenv("PAGER", "less")
	t.Setenv("LESS", "")
	_ = os.Unsetenv("LESS")

	for _, e := range buildChildEnv("less") {
		if strings.HasPrefix(e, "PAGER=") {
			t.Error("PAGER should be removed from child env")
		}
	}
}

func TestBuildChildEnv_SetsLESSDefault(t *testing.T) {
	t.Setenv("LESS", "")
	_ = os.Unsetenv("LESS")

	found := false
	for _, e := range buildChildEnv("less") {
		if e == "LESS=FRX" {
			found = true
		}
	}
	if !found {
		t.Error("expected LESS=FRX when $LESS is unset")
	}
}

func TestBuildChildEnv_PreservesExistingLESS(t *testing.T) {
	t.Setenv("LESS", "R")

	for _, e := range buildChildEnv("less") {
		if e == "LESS=FRX" {
			t.Error("should not override existing $LESS")
		}
	}
}

func TestBuildChildEnv_NoLESSForNonLessPager(t *testing.T) {
	t.Setenv("LESS", "")
	_ = os.Unsetenv("LESS")

	for _, e := range buildChildEnv("more") {
		if strings.HasPrefix(e, "LESS=") {
			t.Error("should not set LESS for non-less pager")
		}
	}
}

func TestPagerWriter_EPIPEWrapped(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()

	pw := &pagerWriter{w: w}
	_, writeErr := pw.Write([]byte("hello"))
	_ = w.Close()

	if writeErr == nil {
		t.Skip("no EPIPE produced on this platform")
	}
	if _, ok := writeErr.(ErrClosedPagerPipe); !ok {
		t.Errorf("expected ErrClosedPagerPipe, got %T: %v", writeErr, writeErr)
	}
}

func openTempFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "pager-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func openPipe(t *testing.T) (*os.File, *os.File, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})
	return r, w, nil
}
