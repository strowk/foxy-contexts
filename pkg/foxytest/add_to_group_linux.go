package foxytest

import (
	"os/exec"
	"syscall"
)

func addToGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
