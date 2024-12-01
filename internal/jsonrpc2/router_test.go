package jsonrpc2

import (
	"fmt"
	"testing"

	"github.com/strowk/foxy-contexts/pkg/mcp"
)

func TestRouter(t *testing.T) {
	r := NewJsonRPCRouter()

	r.SetRequestHandler(&mcp.ListResourcesRequest{},
		func(req Request) (Result, *Error) {
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

	t.Run("Unmarshal list resources and check that result pointer is different every time", func(t *testing.T) {
		data := `{"method":"resources/list","params":{}, "id":1}`
		res := r.Handle([]byte(data))
		if res.Error != nil {
			t.Fatalf("failed: %v", res.Error)
		}
		if res.Result == nil {
			t.Fatalf("result is empty")
		}
		// res is pointer, print the pointer address
		addr1 := fmt.Sprintf("%p", res.Result)

		// marshal and unmarshal to get a new pointer
		res2 := r.Handle([]byte(data))
		if res2.Error != nil {
			t.Fatalf("failed: %v", res2.Error)
		}
		addr2 := fmt.Sprintf("%p", res2.Result)

		if addr1 == addr2 {
			t.Fatalf("expected different pointers, got the same")
		}
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
