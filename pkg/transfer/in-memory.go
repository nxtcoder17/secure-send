package transfer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/nxtcoder17/ivy"
)

type InMemoryTransferManager struct {
	senders map[string]*Sender
	mu      sync.Mutex

	MaxWaitDuration time.Duration
	eventHandlers   []EventHandler
}

// StartTransfer implements TransferManager
func (tm *InMemoryTransferManager) StartTransfer(connectionID string, c *ivy.Context) error {
	sender, ok := tm.senders[connectionID]
	if !ok {
		tm.notify(EventSenderNotFound, "sender not found", "connectionID", connectionID)
		return ivy.NewHTTPError(http.StatusBadRequest, "no sender found")
	}

	sender.Lock()
	defer sender.Unlock()

	go func() {
		<-c.Done()
		slog.Debug("receiver closed", "connectionID", connectionID)
	}()

	// tm.notifySender(sender, EventTransferStarted, fmt.Sprintf("transfer started for connectionID=%v", connectionID))
	tm.notifySender(sender, EventTransferStarted, "transfer started", "connectionID", connectionID)
	defer func() {
		tm.CloseSender(connectionID)
		delete(tm.senders, connectionID)
		tm.notifySender(sender, EventTransferFinished, "transfer finished", "connectionID", connectionID)
	}()

	msg := make([]byte, 1<<8) // []byte of len 256 (2**8)
	transferred := 0

	reader := bufio.NewReader(sender.ReadCloser)
	defer sender.Close()

	for {
		n, err := reader.Read(msg)
		if err != nil {
			if errors.Is(err, io.EOF) {
				if _, err := c.Write(msg[:n]); err != nil {
					tm.notify(EventTransferError, err.Error())
				}
				transferred += n

				c.Flush()
				break
			}

			tm.notifySender(sender, EventTransferError, err.Error())
			slog.Error("while reading file, got", "err", err)
			return nil
		}

		if _, err := c.Write(msg[:n]); err != nil {
			tm.notify(EventTransferError, err.Error())
		}
		transferred += n
		c.Flush()
		tm.notifySender(sender, EventTransferBytesUpdate, "transferred", "bytes", transferred)
	}

	tm.notifySender(sender, EventTransferFinished, "transferred", "bytes", transferred)
	// tm.mu.Lock()
	return nil
}

func (tm *InMemoryTransferManager) Subscribe(onEvent EventHandler) {
	tm.eventHandlers = append(tm.eventHandlers, onEvent)
}

// NewSender implements ConnectionManager.
func (tm *InMemoryTransferManager) NewSender(connectionID string, sender io.ReadCloser) (*Sender, error) {
	if _, ok := tm.senders[connectionID]; ok {
		tm.notify(EventSenderCreationFailed, fmt.Sprintf("sender created with connectionID=%v", connectionID))
		return nil, fmt.Errorf("connectionID already in use, please use another")
	}

	s := Sender{
		connectionID: connectionID,
		ReadCloser:   sender,
	}
	tm.senders[connectionID] = &s

	tm.notifySender(&s, EventSenderCreated, "sender created", "connectionID", connectionID)
	return &s, nil
}

// CloseSender implements TransferManager.
func (tm *InMemoryTransferManager) CloseSender(connectionID string) {
	if sender, ok := tm.senders[connectionID]; ok {
		sender.Close()
	}
	delete(tm.senders, connectionID)
	tm.notify(EventTransferFinished, fmt.Sprintf("sender closed for connection ID (%s)", connectionID))
}

func (tm *InMemoryTransferManager) notify(event Event, msg string, kv ...any) {
	for i := range tm.eventHandlers {
		tm.eventHandlers[i](event, msg, kv...)
	}
}

func (tm *InMemoryTransferManager) notifySender(sender *Sender, event Event, msg string, kv ...any) {
	for i := range sender.subscribers {
		sender.subscribers[i](event, msg, kv...)
	}
	for i := range tm.eventHandlers {
		kv = append(kv, "sender", sender.connectionID)
		tm.eventHandlers[i](event, msg, kv...)
	}
}

func NewInMemoryTransferManager() TransferManager {
	return &InMemoryTransferManager{
		senders:       make(map[string]*Sender),
		eventHandlers: make([]EventHandler, 0, 1),
		mu:            sync.Mutex{},
	}
}
