package sse

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
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
	if len(ev.ID) > 0 {
		_, err := fmt.Fprintf(w, "id: %s\n", ev.ID)
		if err != nil {
			return err
		}
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

func DecodeEvent(r *bufio.Reader) (*Event, error) {
	fieldCaptureRegex := "[\\s]*(data|id|event|retry):[\\s]*([^\\n]+)"
	regexp := regexp.MustCompile(fieldCaptureRegex)
	var ev Event
	readPartialEvent := false

	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				if !readPartialEvent {
					return nil, io.EOF
				}
				break
			}
			return nil, err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			break // end of event
		}

		if bytes.HasPrefix(line, []byte(":")) {
			// comment line
			continue
		}

		if regexp.Match(line) {
			field := regexp.FindSubmatch(line)
			switch string(field[1]) {
			case "data":
				ev.Data = append(ev.Data, field[2]...)
			case "id":
				ev.ID = field[2]
			case "event":
				ev.Event = field[2]
			case "retry":
				ev.Retry = field[2]
			}
		}

		readPartialEvent = true
	}
	if len(ev.Data) == 0 {
		return nil, fmt.Errorf("no data found in event")
	}

	return &ev, nil
}
