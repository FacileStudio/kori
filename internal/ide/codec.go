package ide

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"sync"
)

// Encoder writes events as newline-delimited JSON: one object per line, with
// no literal newline inside one. It serialises its own writes, so two
// goroutines publishing at once cannot interleave halves of two lines.
type Encoder struct {
	mu sync.Mutex
	w  io.Writer
}

// NewEncoder writes events to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode writes one event and the newline that ends it.
func (e *Encoder) Encode(ev Event) error {
	raw, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err = e.w.Write(append(raw, '\n'))
	return err
}

// Decoder reads newline-delimited JSON, one object per line, into whatever
// shape the caller expects: a Command on the session's side, an Event on an
// editor's.
type Decoder struct {
	r *bufio.Reader
}

// NewDecoder reads objects from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

// Decode reads the next object into v. Unknown fields are ignored, which is
// what lets either side add one without a version bump, and io.EOF is what a
// closed connection ends with.
func (d *Decoder) Decode(v any) error {
	line, err := d.line()
	if err != nil {
		return err
	}
	return json.Unmarshal(line, v)
}

// line returns the next line that carries an object, skipping the blank ones
// a writer might leave between them.
func (d *Decoder) line() ([]byte, error) {
	for {
		line, err := d.r.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			return line, nil
		}
		if err != nil {
			return nil, err
		}
	}
}
