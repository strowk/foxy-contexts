package sse

import (
	"bytes"
	"fmt"
	"io"
)

type Event struct {
	ID    []byte
	Data  []byte
	Event []byte
	Retry []byte
}

func (ev *Event) MarshalTo(w io.Writer) error {
	if len(ev.Data) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(w, "id: %s\n", ev.ID)
	if err != nil {
		return err
	}
	sd := bytes.Split(ev.Data, []byte("\n"))
	for i := range sd {
		if _, err := fmt.Fprintf(w, "data: %s\n", sd[i]); err != nil {
			return err
		}
	}

	if len(ev.Event) > 0 {
		if _, err := fmt.Fprintf(w, "event: %s\n", ev.Event); err != nil {
			return err
		}
	}

	if len(ev.Retry) > 0 {
		if _, err := fmt.Fprintf(w, "retry: %s\n", ev.Retry); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(w, "\n"); err != nil {
		return err
	}

	return nil
}

type CommentEvent struct {
	Comment []byte
}

func (ev *CommentEvent) MarshalTo(w io.Writer) error {
	if len(ev.Comment) == 0 {
		return nil
	}

	if _, err := fmt.Fprintf(w, ": %s\n\n", ev.Comment); err != nil {
		return err
	}

	return nil
}
