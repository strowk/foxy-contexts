package foxytest

import (
	"os"
	"syscall"
)

func stop(process *os.Process) error {
	return syscall.Kill(-process.Pid, syscall.SIGTERM)
}
