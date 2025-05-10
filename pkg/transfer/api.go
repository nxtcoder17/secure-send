package transfer

import (
	"io"

	"github.com/nxtcoder17/ivy"
)

type Connection struct {
	Sender   io.ReadCloser
	Receiver io.Writer
}

type Event int

const (
	EventUnknownState Event = 0

	EventSenderCreated        = 1
	EventSenderCreationFailed = 2
	EventSenderNotFound       = 3

	EventReceiverCreated = 4

	EventTransferStarted     = 5
	EventTransferBytesUpdate = 6
	EventTransferError       = 7
	EventTransferFinished    = 8
)

func (ev Event) String() string {
	switch ev {
	case EventSenderCreated:
		return "EventSenderCreated"
	case EventSenderNotFound:
		return "EventSenderNotFound"
	case EventReceiverCreated:
		return "EventReceiverCreated"
	case EventTransferStarted:
		return "EventTransferStarted"
	case EventTransferBytesUpdate:
		return "EventTransferBytesUpdate"
	case EventTransferError:
		return "EventTransferError"
	case EventTransferFinished:
		return "EventTransferFinished"
	default:
		return "EventUnknownState"
	}
}

type EventHandler func(event Event, msg string, kv ...any)

type TransferManager interface {
	NewSender(connectionID string, sender io.ReadCloser) (*Sender, error)
	CloseSender(connectionID string)
	StartTransfer(connectionID string, c *ivy.Context) error
	Subscribe(onEvent EventHandler)
}
