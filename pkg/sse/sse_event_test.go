package sse

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

const body = `data: {"jsonrpc":"2.0","result":{"tools":[{"description":"The great tool","inputSchema":{"type":"object"},"name":"my-great-tool"}]},"id":1}

data: {"jsonrpc":"2.0","result":{"prompts":[]},"id":2}

`

func TestDecodeEvent(t *testing.T) {
	body := []byte(body)
	rd := bufio.NewReader(bytes.NewBuffer(body))

	for i := 0; i < 2; i++ {
		ev, err := DecodeEvent(rd)
		if err != nil {
			t.Fatalf("failed to decode event: %v", err)
		}

		if ev.Data == nil {
			t.Fatalf("expected id and data to be set")
		}
		if len(ev.Data) == 0 {
			t.Fatalf("expected data to be set")
		}

		if i == 0 {
			assert.Equal(t, []byte(`{"jsonrpc":"2.0","result":{"tools":[{"description":"The great tool","inputSchema":{"type":"object"},"name":"my-great-tool"}]},"id":1}`), ev.Data)
		} else {
			assert.Equal(t, []byte(`{"jsonrpc":"2.0","result":{"prompts":[]},"id":2}`), ev.Data)
		}
	}

}
