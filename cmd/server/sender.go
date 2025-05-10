package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nxtcoder17/ivy"
	"github.com/nxtcoder17/secure-send/pkg/transfer"
)

type SendParams struct {
	TransferManager transfer.TransferManager
	ConnectionID    string
	MaxWaitDuration time.Duration
}

func Send(c *ivy.Context, params SendParams) error {
	sender, err := params.TransferManager.NewSender(params.ConnectionID, c.Body())
	if err != nil {
		return err
	}

	validityCtx, cf := context.WithTimeout(c, params.MaxWaitDuration)
	defer cf()

	started := make(chan bool, 1)
	finished := make(chan bool, 1)

	go func() {
		sender.Subscribe(func(event transfer.Event, msg string, kv ...any) {
			switch event {
			case transfer.EventTransferStarted:
				started <- true
			case transfer.EventTransferBytesUpdate:
				for i := 0; i <= len(kv); i += 2 {
					if kv[i].(string) == "bytes" {
						value := kv[i+1].(int)
						switch {
						case value < 1024*1024:
							fmt.Fprintf(c, "transferred %.2f KBs\r\n", float64(value)/1024)
						default:
							fmt.Fprintf(c, "transferred %.2f MBs\r\n", float64(value)/1024/1024)
						}
						break
					}
				}
			case transfer.EventTransferError:
				finished <- true
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
