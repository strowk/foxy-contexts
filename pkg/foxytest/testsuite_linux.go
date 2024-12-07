package foxytest

import (
	"os"
	"syscall"
)

func stop(process *os.Process) error {
	return process.Signal(syscall.SIGTERM)
}
