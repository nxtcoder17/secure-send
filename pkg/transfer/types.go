package transfer

import (
	"fmt"
	"io"
)

type Sender struct {
	io.ReadCloser
	closed bool
}

func (r *Sender) Read(p []byte) (int, error) {
	return r.ReadCloser.Read(p)
}

func (r *Sender) Close() error {
	if r.closed {
		return fmt.Errorf("already closed")
	}
	r.closed = true
	return r.ReadCloser.Close()
}

func (r *Sender) IsClosed() bool {
	return r.closed
}
