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
	ts.WithExecutable("go", []string{"run", "main.go"})
	ts.WithTransport(foxytest.NewTestTransportStreamableHTTP("http://localhost:80/mcp"))
	ts.WithLogging()
	cntrl := foxytest.NewTestRunner(t)
	ts.Run(cntrl)
	ts.AssertNoErrors(cntrl)
}
