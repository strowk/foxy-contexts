package main

import (
	"testing"

	"github.com/strowk/foxy-contexts/pkg/foxytest"
)

func TestWithFoxytest(t *testing.T) {
	ts, err := foxytest.Read("testdata")
	if err != nil {
		t.Fatal(err)
	}
	// ts.WithExecutable("list_current_dir_files_tool", []string{})
	ts.WithExecutable("go", []string{"run", "main.go"})
	cntrl := foxytest.NewTestRunner(t)
	ts.Run(cntrl)
	ts.AssertNoErrors(cntrl)
}
