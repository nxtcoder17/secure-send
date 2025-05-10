package transfer

import (
	"io"
	"sync"
)

type Sender struct {
	io.ReadCloser
	connectionID string
	sync.Mutex
	subscribers []EventHandler
}

func (r *Sender) Read(p []byte) (int, error) {
	return r.ReadCloser.Read(p)
}

func (r *Sender) Close() error {
	return r.ReadCloser.Close()
}

func (r *Sender) Subscribe(onEvent EventHandler) {
	r.subscribers = append(r.subscribers, onEvent)
}

type Receiver struct {
	io.Writer
	connectionID string
}
