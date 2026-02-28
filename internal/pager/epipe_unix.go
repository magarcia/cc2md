//go:build !windows

package pager

import (
	"errors"
	"syscall"
)

func isEPIPE(err error) bool {
	return errors.Is(err, syscall.EPIPE)
}
