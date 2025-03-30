package jsonrpc2

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/session"
)

func testContext() context.Context {
	return session.WithNewSession(context.Background())
}

func TestRouter(t *testing.T) {
	r := NewJsonRPCRouter()

	r.SetRequestHandler(&mcp.ListResourcesRequest{},
		func(ctx context.Context, req Request) (Result, *Error) {
			return &mcp.ListResourcesResult{
				Resources: []mcp.Resource{
					{
						Name: "resource1",
						Uri:  "uri1",
					},
				},
			}, nil
		},
	)

	r.SetNotificationHandler(&mcp.InitializedNotification{}, func(ctx context.Context, req Request) {})

	t.Run("Handle list resources and check that result pointer is different every time", func(t *testing.T) {
		data := `{"method":"resources/list","params":{}, "id":1}`
		responses := r.Handle(testContext(), []byte(data))
		require.Len(t, responses, 1)
		res := responses[0]
		if res.Error != nil {
			t.Fatalf("failed: %v", res.Error)
		}
		if res.Result == nil {
			t.Fatalf("result is empty")
		}
		// res is pointer, print the pointer address
		addr1 := fmt.Sprintf("%p", res.Result)

		// marshal and unmarshal to get a new pointer
		responses2 := r.Handle(testContext(), []byte(data))
		require.Len(t, responses2, 1)
		res2 := responses2[0]
		if res2.Error != nil {
			t.Fatalf("failed: %v", res2.Error)
		}
		addr2 := fmt.Sprintf("%p", res2.Result)

		if addr1 == addr2 {
			t.Fatalf("expected different pointers, got the same")
		}
	})

	t.Run("Handle initialized notification", func(t *testing.T) {
		data := `{"method":"notifications/initialized","params":{}}`
		responses := r.Handle(testContext(), []byte(data))
		require.Len(t, responses, 1)
		res := responses[0]
		require.Nil(t, res)
	})

	t.Run("Handle empty request", func(t *testing.T) {
		data := `{}`
		responses := r.Handle(testContext(), []byte(data))
		require.Len(t, responses, 1)
		res := responses[0]
		if res.Error == nil {
			t.Fatalf("expected error, got nil")
		}
		if res.Error.Code != -32600 {
			t.Fatalf("expected code -32600, got %d", res.Error.Code)
		}
	})

	t.Run("Handle unparseable request", func(t *testing.T) {
		data := `not a json`
		responses := r.Handle(testContext(), []byte(data))
		require.Len(t, responses, 1)
		res := responses[0]
		if res.Error == nil {
			t.Fatalf("expected error, got nil")
		}
		assert.Equal(t, -32700, res.Error.Code)
		assert.Equal(t, "Parse error", res.Error.Message)
		assert.Equal(t, "invalid character 'o' in literal null (expecting 'u')", res.Error.Data)
	})

	t.Run("Handle null request", func(t *testing.T) {
		data := `null`
		responses := r.Handle(testContext(), []byte(data))
		require.Len(t, responses, 1)
		res := responses[0]
		if res.Error == nil {
			t.Fatalf("expected error, got nil")
		}
		assert.Equal(t, -32600, res.Error.Code)
		assert.Equal(t, "Invalid Request", res.Error.Message)
		assert.Equal(t, "Request is null, but must be an object", res.Error.Data)
	})

	t.Run("Handle unknown method", func(t *testing.T) {
		data := `{"method":"unknown","params":{}, "id":1}`
		responses := r.Handle(testContext(), []byte(data))
		require.Len(t, responses, 1)
		res := responses[0]
		if res.Error == nil {
			t.Fatalf("expected error, got nil")
		}
		assert.Equal(t, -32601, res.Error.Code)
		assert.Equal(t, "Method not found", res.Error.Message)
		assert.Equal(t, "request for method unknown not found in registry", res.Error.Data)
	})

	t.Run("Handle invalid method type", func(t *testing.T) {
		data := `{"method":1,"params":{}, "id":1}`
		responses := r.Handle(testContext(), []byte(data))
		require.Len(t, responses, 1)
		res := responses[0]
		if res.Error == nil {
			t.Fatalf("expected error, got nil")
		}
		assert.Equal(t, -32600, res.Error.Code)
		assert.Equal(t, "field method in request must be a string, but got float64", res.Error.Data)
	})
}

func TestMarshal(t *testing.T) {
	t.Run("Marshal list resources", func(t *testing.T) {
		res := &mcp.ListResourcesResult{
			Resources: []mcp.Resource{
				{
					Name: "resource1",
					Uri:  "uri1",
				},
			},
		}
		data, err := Marshal(RequestId{IdNumber: 1, IdIsNum: true}, res, nil)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		expected := `{"jsonrpc":"2.0","result":{"resources":[{"name":"resource1","uri":"uri1"}]},"id":1}`
		if string(data) != expected {
			t.Fatalf("expected %s, got %s", expected, string(data))
		}
	})
}
