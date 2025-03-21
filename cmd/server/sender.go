package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nxtcoder17/ivy"
	"github.com/nxtcoder17/secure-send/pkg/transfer"
)

// const tokenChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
//
// func RandStringBytes(n int) string {
// 	b := make([]byte, n)
// 	for i := range b {
// 		b[i] = tokenChars[rand.IntN(len(tokenChars))]
// 	}
// 	return string(b)
// }

type SendParams struct {
	transfer.TransferManager
	ConnectionID    string
	MaxWaitDuration time.Duration
}

func Send(c *ivy.Context, params SendParams) error {
	if err := params.TransferManager.NewSender(params.ConnectionID, c.Body()); err != nil {
		return err
	}

	validityCtx, cf := context.WithTimeout(c, params.MaxWaitDuration)
	defer cf()

	started := make(chan bool, 1)
	finished := make(chan bool, 1)

	go func() {
		params.TransferManager.Subscribe(func(event transfer.Event, msg string) {
			switch event {
			case transfer.EventTransferStarted:
				started <- true
			case transfer.EventTransferBytesUpdate:
				c.Write([]byte("\r" + msg))
			case transfer.EventTransferFinished:
				finished <- true
			}
		})
	}()

	select {
	case <-validityCtx.Done():
		return fmt.Errorf("timed out waiting for receiver for this file")
	case <-started:
	}

	<-finished

	return nil
}
