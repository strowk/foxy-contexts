package main

import (
	"os"
	"testing"

	"github.com/strowk/foxy-contexts/pkg/foxytest"
)

func TestListContexts(t *testing.T) {
	ts, err := foxytest.Read("testdata")
	if err != nil {
		t.Fatal(err)
	}
	os.Setenv("KUBECONFIG", "./testdata/kubeconfig")
	ts.WithExecutable("go", []string{"run", "."})
	cntrl := foxytest.NewTestRunner(t)
	ts.Run(cntrl)
	ts.AssertNoErrors(cntrl)
}
