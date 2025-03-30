package main

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestStartStop(t *testing.T) {
	cmd := exec.Command("go", "run", "lifecycle.go")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	processStopped := make(chan struct{})
	go func() {
		defer close(processStopped)
		cmd.Run()
	}()

	// this should cause process to exit
	err = stdin.Close()
	if err != nil {
		t.Error(err)
		return
	}

	timer := time.NewTimer(10 * time.Second)
	select {
	case <-processStopped:
		// process exited
	case <-timer.C:
		t.Error("process did not exit in time")
	}
}
