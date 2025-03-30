package jsonrpc2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

type Result any

type Request interface {
	GetMethod() string
}

type JsonRpcRouter interface {
	/// SetRequestHandler sets a handler for a request that would return a response
	SetRequestHandler(request Request, handler func(ctx context.Context, req Request) (Result, *Error))

	/// SetNotificationHandler sets a handler for a notification, that would not return any response
	SetNotificationHandler(request Request, handler func(ctx context.Context, req Request))

	/// Handle processes incoming JSON-RPC request and either returns an array of
	// JSON-RPC results or error responses or nil if successfully processed notification
	Handle(ctx context.Context, b []byte) []*JsonRpcResponse
}

type RequestId struct {
	IdString    string
	IdNumber    int
	IdIsNum     bool
	IdIsNull    bool
	IdIsMissing bool
}

func NewStringRequestId(id string) RequestId {
	return RequestId{
		IdString: id,
	}
}

func NewIntRequestId(id int) RequestId {
	return RequestId{
		IdNumber: id,
		IdIsNum:  true,
	}
}

func NewMissingRequestId() RequestId {
	return RequestId{
		IdIsMissing: true,
	}
}

func (r RequestId) MarshalJSON() ([]byte, error) {
	if r.IdIsMissing {
		return nil, fmt.Errorf("id is missing, this is not supposed to be marshaled")
	}
	if r.IdIsNull {
		return json.Marshal(nil)
	}
	if r.IdIsNum {
		return json.Marshal(r.IdNumber)
	}
	return json.Marshal(r.IdString)
}

func NewNullRequestId() RequestId {
	return RequestId{
		IdIsNull: true,
	}
}

type router struct {
	requestHandlers      map[string]func(ctx context.Context, req Request) (Result, *Error)
	notificationHandlers map[string]func(ctx context.Context, req Request)
	requestRegistry      map[string]func() Request
}

func NewJsonRPCRouter() JsonRpcRouter {
	return &router{
		requestHandlers:      map[string]func(ctx context.Context, req Request) (Result, *Error){},
		requestRegistry:      map[string]func() Request{},
		notificationHandlers: map[string]func(ctx context.Context, req Request){},
	}
}

func (r *router) saveRequestToRegistry(method string, request Request) {
	r.requestRegistry[method] = func() Request {
		// copy using reflect, note that request is expected to be a pointer
		requestCopy := reflect.New(reflect.TypeOf(request).Elem()).Interface().(Request)
		return requestCopy
	}
}

func (r *router) SetRequestHandler(request Request, handler func(ctx context.Context, req Request) (Result, *Error)) {
	method := request.GetMethod()
	r.saveRequestToRegistry(method, request)
	r.requestHandlers[method] = handler
}

func (r *router) SetNotificationHandler(request Request, handler func(ctx context.Context, req Request)) {
	method := request.GetMethod()
	r.saveRequestToRegistry(method, request)
	r.notificationHandlers[method] = handler
}

func (r *router) getRequestHandler(method string) func(ctx context.Context, req Request) (Result, *Error) {
	if handler, ok := r.requestHandlers[method]; ok {
		return handler
	}
	return nil
}

func (r *router) getNotificationHandler(method string) func(ctx context.Context, req Request) {
	if handler, ok := r.notificationHandlers[method]; ok {
		return handler
	}
	return nil
}

func (r *router) handle(
	ctx context.Context,
	buf []byte,
	method string,
	id RequestId,
) (Result, RequestId, *Error) {
	if regEntry, ok := r.requestRegistry[method]; ok {
		req := regEntry()
		err := json.Unmarshal(buf, req)
		if err != nil {
			return nil, id, &Error{
				Code:    -32700,
				Message: "Parse error",
				Data:    err.Error(),
			}
		}

		if id.IdIsMissing {
			handler := r.getNotificationHandler(method)
			if handler == nil {
				return nil, NewNullRequestId(), methodNotFound(fmt.Sprintf("handler for method %v not found to process notification", method))
			}
			handler(ctx, req)
			return nil, id, nil
		} else {
			handler := r.getRequestHandler(method)
			if handler == nil {
				return nil, id, methodNotFound(fmt.Sprintf("handler for method %v not found to process request", method))
			}

			res, err := handler(ctx, req)
			if err != nil {
				return nil, id, err
			}
			return res, id, nil
		}

	} else {
		return nil, id, methodNotFound(fmt.Sprintf("request for method %v not found in registry", method))
	}
}

func getId(raw map[string]interface{}) (*RequestId, error) {
	if idField, ok := raw["id"]; ok {
		if idField == nil {
			return nil, fmt.Errorf("field id in request is required cannot be null")
		}
		if idString, ok := idField.(string); ok {
			return &RequestId{
				IdString: idString,
			}, nil
		} else if idNumber, ok := idField.(float64); ok {
			return &RequestId{
				IdNumber: int(idNumber),
				IdIsNum:  true,
			}, nil
		} else {
			return nil, fmt.Errorf("field id in request is expected to be string or int, but got %v", reflect.TypeOf(idField))
		}
	} else {
		// this means that we received is a notification and response is not required
		// besides we need to use use notification handlers
		return &RequestId{
			IdIsMissing: true,
		}, nil
	}
}

func (r *router) Handle(ctx context.Context, buf []byte) []*JsonRpcResponse {
	trimmedBytes := bytes.TrimLeft(buf, " \t\r\n")

	isArray := len(trimmedBytes) > 0 && trimmedBytes[0] == '['
	isObject := len(trimmedBytes) > 0 && trimmedBytes[0] == '{'

	if isArray {
		var rawArray []json.RawMessage
		if err := json.Unmarshal(buf, &rawArray); err != nil {
			return errResponseWithNullId(parseError(err.Error()))
		}
		var resp []*JsonRpcResponse
		for _, raw := range rawArray {
			resp = append(resp, getResponse(r.handleSingleObject(ctx, raw)))
		}
		return resp
	}

	if isObject {
		return getResponses(r.handleSingleObject(ctx, buf))
	}

	// need to check whether request is correct json at all
	var raw any
	if err := json.Unmarshal(buf, &raw); err != nil {
		return errResponseWithNullId(parseError(err.Error()))
	}

	// finally if it was correct json, but probably not an object or array, we need to return invalid request response
	if raw == nil {
		return errResponseWithNullId(invalidRequest("Request is null, but must be an object"))
	}

	return errResponseWithNullId(invalidRequest(fmt.Sprintf("Request is expected to be an object or array, but was %T' : %s", raw, string(trimmedBytes))))
}

func (r *router) handleSingleObject(ctx context.Context, raw json.RawMessage) (Result, RequestId, *Error) {
	if raw == nil {
		return nil, NewNullRequestId(), invalidRequest("Request is null, but must be an object")
	}

	var rawMap map[string]any
	err := json.Unmarshal(raw, &rawMap)
	if err != nil {
		return nil, NewNullRequestId(), parseError(err.Error())
	}

	id, err := getId(rawMap)
	if err != nil {
		return nil, NewNullRequestId(), invalidRequest(err.Error())
	}

	if method, ok := rawMap["method"]; ok {
		if method == nil {
			return nil, *id, invalidRequest("Method is required, but was null")
		}

		if methodString, ok := method.(string); ok {
			res, resId, err := r.handle(ctx, raw, methodString, *id)
			if err != nil {
				return nil, resId, err
			}
			return res, resId, nil
		} else {
			return nil, *id, invalidRequest(fmt.Sprintf("field method in request must be a string, but got %v", reflect.TypeOf(method)))
		}

	} else {
		return nil, *id, invalidRequest("Method is required, but is missing")
	}
}

func getResponse(res Result, id RequestId, err *Error) *JsonRpcResponse {
	if err != nil {
		return &JsonRpcResponse{
			Id:    id,
			Error: err,
		}
	}
	if res != nil {
		return &JsonRpcResponse{
			Id:     id,
			Result: &res,
		}
	}
	return nil
}

func getResponses(res Result, id RequestId, err *Error) []*JsonRpcResponse {
	return []*JsonRpcResponse{getResponse(res, id, err)}
}

func errResponseWithNullId(err *Error) []*JsonRpcResponse {
	return getResponses(nil, NewNullRequestId(), err)
}
