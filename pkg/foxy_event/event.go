package foxyevent

type Event interface {
	event()
}

type SSEClientConnected struct {
	ClientIP string
}

func (SSEClientConnected) event() {}

type SSEClientDisconnected struct {
	ClientIP string
}

func (SSEClientDisconnected) event() {}

type SSEFailedCreatingEvent struct {
	Err error
}

func (SSEFailedCreatingEvent) event() {}

type SSEFailedMarshalEvent struct {
	Err error
}

func (SSEFailedMarshalEvent) event() {}

type StreamingHTTPFailedMarshalEvent struct {
	Err error
}

func (StreamingHTTPFailedMarshalEvent) event() {}

type StdioFailedMarhalResponse struct {
	Err error
}

func (StdioFailedMarhalResponse) event() {}

type StdioSendingResponse struct {
	Data []byte
}

func (StdioSendingResponse) event() {}

type StdioFailedReadingInput struct {
	Err error
}

func (StdioFailedReadingInput) event() {}

type StdioFailedWriting struct {
	Err error
}

func (StdioFailedWriting) event() {}

type FailedCreatingSession struct {
	Err error
}

func (FailedCreatingSession) event() {}
