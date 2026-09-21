//go:build windows

package updatecheck

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// installAndRestart stages newBin next to the running executable and hands off
// to a detached cmd script that waits for this process to exit, swaps the
// binary and starts it again. Returns an error if staging fails; on success the
// caller should exit the process.
func installAndRestart(newBin string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	staged := exe + ".new"
	if err := copyFile(newBin, staged); err != nil {
		return fmt.Errorf("cannot stage new binary: %w", err)
	}

	// Preserve the original arguments and working directory — a bare
	// `start "" exe` would lose e.g. `-c conf.ini` on restart.
	var args strings.Builder
	for _, a := range os.Args[1:] {
		fmt.Fprintf(&args, ` "%s"`, strings.ReplaceAll(a, `"`, ""))
	}
	cwd, _ := os.Getwd()

	// Helper waits for our PID to exit, swaps the binary, restarts it, then
	// deletes itself.
	script := exe + ".update.cmd"
	content := fmt.Sprintf(`@echo off
:wait
tasklist /FI "PID eq %d" 2>nul | find "%d" >nul
if %%errorlevel%%==0 (timeout /t 1 /nobreak >nul & goto wait)
move /y "%s" "%s" >nul
cd /d "%s"
start "" "%s"%s
del "%s"
`, os.Getpid(), os.Getpid(), staged, exe, cwd, exe, args.String(), script)
	if err := os.WriteFile(script, []byte(content), 0o600); err != nil {
		return fmt.Errorf("cannot write update helper: %w", err)
	}

	cmd := exec.Command("cmd", "/C", "start", "", "/min", script)
	cmd.Dir = filepath.Dir(exe)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start update helper: %w", err)
	}
	return nil
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

// unameMachine is unused on Windows (arm32 is not a supported server target).
func unameMachine() (string, error) {
	return "", fmt.Errorf("not supported")
}
