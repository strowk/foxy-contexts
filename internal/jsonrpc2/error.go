package jsonrpc2

import "fmt"

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func invalidRequest(data string) *Error {
	return &Error{
		Code:    -32600,
		Message: "Invalid Request",
		Data:    data,
	}
}

func methodNotFound(data string) *Error {
	return &Error{
		Code:    -32601,
		Message: "Method not found",
		Data:    data,
	}
}

func parseError(data string) *Error {
	return &Error{
		Code:    -32700,
		Message: "Parse error",
		Data:    data,
	}
}

func NewServerError(
	code int,
	data interface{},
) *Error {
	if code > -32000 || code < -32099 {
		panic(fmt.Errorf("server error code must be between -32000 and -32099, but got %d", code))
	}
	return &Error{
		Code:    code,
		Message: "Server error",
		Data:    data,
	}
}

func NewAppError(
	code int,
	message string,
	data interface{},
) *Error {
	if code < -32000 {
		panic(fmt.Errorf("application error code must be bigger than -32000, but got %d", code))
	}
	return &Error{
		Code:    code,
		Message: message,
		Data:    data,
	}
}
