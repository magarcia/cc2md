package pager

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ErrClosedPagerPipe wraps an EPIPE from writing to a closed pager stdin.
// Callers should treat this as a clean exit (user quit the pager).
type ErrClosedPagerPipe struct {
	error
}

type pagerWriter struct {
	w io.WriteCloser
}

func (pw *pagerWriter) Write(p []byte) (int, error) {
	n, err := pw.w.Write(p)
	if err != nil && isEPIPE(err) {
		return n, ErrClosedPagerPipe{err}
	}
	return n, err
}

type Pager struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

// Start launches a pager if out is a TTY. Returns nil Pager and direct writer
// on non-TTY, missing binary, or PAGER=cat.
func Start(out *os.File) (*Pager, io.Writer, error) {
	if !isTTY(out) {
		return nil, out, nil
	}

	pagerCmd := os.Getenv("PAGER")
	if pagerCmd == "" {
		pagerCmd = "less"
	}
	if pagerCmd == "cat" {
		return nil, out, nil
	}

	fields := strings.Fields(pagerCmd)
	bin, err := exec.LookPath(fields[0])
	if err != nil {
		return nil, out, nil
	}

	cmd := exec.Command(bin, fields[1:]...)
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	cmd.Env = buildChildEnv(pagerCmd)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, out, fmt.Errorf("creating pager stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, out, nil
	}

	return &Pager{cmd: cmd, stdin: stdin}, &pagerWriter{w: stdin}, nil
}

func (p *Pager) Stop() {
	_ = p.stdin.Close()
	_ = p.cmd.Wait()
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// buildChildEnv removes PAGER (prevent recursion) and sets LESS=FRX if unset.
func buildChildEnv(pagerCmd string) []string {
	env := os.Environ()
	out := make([]string, 0, len(env)+1)

	hasLess := false
	for _, e := range env {
		if strings.HasPrefix(e, "PAGER=") {
			continue
		}
		if strings.HasPrefix(e, "LESS=") {
			hasLess = true
		}
		out = append(out, e)
	}

	if !hasLess && strings.Contains(pagerCmd, "less") {
		out = append(out, "LESS=FRX")
	}

	return out
}
