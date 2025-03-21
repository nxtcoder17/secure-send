package transfer

import "io"

type Connection struct {
	Sender   io.ReadCloser
	Receiver io.Writer
}

type Event int

const (
	EventUnknownState Event = 0

	EventSenderCreated  = 1
	EventSenderNotFound = 2

	EventReceiverCreated = 3

	EventTransferStarted     = 4
	EventTransferBytesUpdate = 5
	EventTransferError       = 6
	EventTransferFinished    = 7
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

type EventHandler func(event Event, msg string)

type TransferManager interface {
	NewSender(connectionID string, sender io.ReadCloser) error
	StartTransfer(connectionID string, writer io.Writer) error
	Subscribe(onEvent EventHandler)
}
