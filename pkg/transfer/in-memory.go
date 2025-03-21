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
)

type InMemoryTransferManager struct {
	senders   map[string]Sender
	receivers map[string]io.Writer
	mu        sync.Mutex

	MaxWaitDuration time.Duration
	eventHandlers   []EventHandler
}

// StartTransfer implements TransferManager
func (tm *InMemoryTransferManager) StartTransfer(connectionID string, writer io.Writer) error {
	sender, ok := tm.senders[connectionID]
	if !ok {
		tm.notify(EventSenderNotFound, fmt.Sprintf("sender not found for connectionID=%v", connectionID))
		return fmt.Errorf("no sender found")
	}

	tm.notify(EventTransferStarted, fmt.Sprintf("transfer started for connectionID=%v", connectionID))

	msg := make([]byte, 1<<8) // []byte of len 256 (2**8)
	transferred := 0

	reader := bufio.NewReader(sender.ReadCloser)
	defer sender.Close()

	for {
		n, err := reader.Read(msg)
		if err != nil {
			if errors.Is(err, io.EOF) {
				if _, err := writer.Write(msg[:n]); err != nil {
					tm.notify(EventTransferError, err.Error())
				}
				transferred += n

				if flusher, ok := writer.(http.Flusher); ok {
					flusher.Flush()
				}

				tm.notify(EventTransferBytesUpdate, fmt.Sprintf("written %.2f MBs (%d bytes)", float64(transferred)/1024/1024, transferred))
				break
			}

			tm.notify(EventTransferError, err.Error())
			slog.Error("while reading file, got", "err", err)
			return nil
		}

		if _, err := writer.Write(msg[:n]); err != nil {
			tm.notify(EventTransferError, err.Error())
		}
		transferred += n
		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}

		tm.notify(EventTransferBytesUpdate, fmt.Sprintf("written %.2f MBs (%d bytes)", float64(transferred)/1024/1024, transferred))
		// <-time.After(500 * time.Millisecond)
	}

	tm.notify(EventTransferFinished, fmt.Sprintf("written %.2f MBs (%d bytes)", float64(transferred)/1024/1024, transferred))
	tm.mu.Lock()
	delete(tm.senders, connectionID)
	delete(tm.receivers, connectionID)
	return nil
}

func (tm *InMemoryTransferManager) Subscribe(onEvent EventHandler) {
	tm.eventHandlers = append(tm.eventHandlers, onEvent)
}

// NewSender implements ConnectionManager.
func (tm *InMemoryTransferManager) NewSender(connectionID string, sender io.ReadCloser) error {
	if _, ok := tm.senders[connectionID]; ok {
		return fmt.Errorf("connectionID already in use, please use another")
	}
	tm.senders[connectionID] = Sender{
		ReadCloser: sender,
		closed:     false,
	}

	tm.notify(EventSenderCreated, fmt.Sprintf("sender created with connectionID=%v", connectionID))
	return nil
}

func (tm *InMemoryTransferManager) notify(event Event, msg string) {
	for i := range tm.eventHandlers {
		tm.eventHandlers[i](event, msg)
	}
}

func NewInMemoryTransferManager() TransferManager {
	return &InMemoryTransferManager{
		senders:       make(map[string]Sender),
		receivers:     make(map[string]io.Writer),
		eventHandlers: make([]EventHandler, 0, 1),
		mu:            sync.Mutex{},
	}
}
