package jsonrpc2

import "encoding/json"

type JsonRpcResponse struct {
	Id     RequestId
	Result *Result
	Error  *Error
}

func Marshal(id RequestId, v Result, e *Error) ([]byte, error) {
	if e != nil {
		return json.Marshal(struct {
			JsonRpc string    `json:"jsonrpc"`
			Error   *Error    `json:"error"`
			Id      RequestId `json:"id"`
		}{
			JsonRpc: "2.0",
			Error:   e,
			Id:      id,
		})
	}
	return json.Marshal(struct {
		JsonRpc string    `json:"jsonrpc"`
		Result  Result    `json:"result"`
		Id      RequestId `json:"id"`
	}{
		JsonRpc: "2.0",
		Result:  v,
		Id:      id,
	})
}
