package foxytest

import (
	"os"
	"os/exec"
	"strconv"
)

func stop(process *os.Process) error {
	// Note: I tried to use this: https://github.com/iwdgo/sigintwindows/blob/5b9fdf930454cd8036eb5c8997763d9fe61d637d/signal_windows.go#L14-L26
	// , however this kills Git Bash terminal as well, not just the underlying process.
	// There is no issue with PowerShell terminal though, so it seems that using taskkill
	// is the only safe enough way at the moment.
	taskKill := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(process.Pid))
	// taskKill.Stdout = os.Stdout
	taskKill.Stderr = os.Stderr
	return taskKill.Run()
}
