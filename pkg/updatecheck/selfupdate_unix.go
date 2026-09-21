//go:build !windows

package updatecheck

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// installAndRestart swaps the running binary for newBin, then re-executes the
// process image in place — no supervisor needed. On success it never returns;
// on failure the original binary is restored and an error is returned.
func installAndRestart(newBin string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	bak := exe + ".bak"
	_ = os.Remove(bak)
	if err := os.Rename(exe, bak); err != nil {
		return fmt.Errorf("cannot move aside current binary: %w", err)
	}
	if err := os.Rename(newBin, exe); err != nil {
		// rename across filesystems can fail — fall back to copy
		if err := copyFile(newBin, exe); err != nil {
			_ = os.Rename(bak, exe)
			return fmt.Errorf("cannot install new binary: %w", err)
		}
	}
	_ = os.Chmod(exe, 0o755)

	if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil {
		// exec failed — restore the old binary and report
		_ = os.Remove(exe)
		_ = os.Rename(bak, exe)
		return fmt.Errorf("re-exec failed: %w", err)
	}
	return nil // unreachable
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func unameMachine() (string, error) {
	out, err := exec.Command("uname", "-m").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
