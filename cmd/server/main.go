package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/nxtcoder17/go.pkgs/log"
	"github.com/nxtcoder17/ivy"
	"github.com/nxtcoder17/secure-send/pkg/transfer"
)

var BuiltAt string

func init() {
	if BuiltAt == "" {
		BuiltAt = time.Now().Format(time.RFC822)
	}
}

func main() {
	addr := flag.String("addr", ":3000", "--addr [host]:<port>")
	maxWait := flag.String("max-wait", "120s", "--max-wait <duration>")
	flag.Parse()

	maxWaitDuration, err := time.ParseDuration(*maxWait)
	if err != nil {
		panic(err)
	}

	logger := log.New()

	fmt.Printf(`
 ▗▄▄▖▗▄▄▄▖ ▗▄▄▖▗▖ ▗▖▗▄▄▖ ▗▄▄▄▖   ▗▄▄▖▗▄▄▄▖▗▖  ▗▖▗▄▄▄ 
▐▌   ▐▌   ▐▌   ▐▌ ▐▌▐▌ ▐▌▐▌     ▐▌   ▐▌   ▐▛▚▖▐▌▐▌  █
 ▝▀▚▖▐▛▀▀▘▐▌   ▐▌ ▐▌▐▛▀▚▖▐▛▀▀▘   ▝▀▚▖▐▛▀▀▘▐▌ ▝▜▌▐▌  █
▗▄▄▞▘▐▙▄▄▖▝▚▄▄▖▝▚▄▞▘▐▌ ▐▌▐▙▄▄▖  ▗▄▄▞▘▐▙▄▄▖▐▌  ▐▌▐▙▄▄▀

› Built on %s
› Http Server running on %s
`, BuiltAt, *addr)

	tm := transfer.NewInMemoryTransferManager()

	tm.Subscribe(func(event transfer.Event, msg string) {
		switch event {
		case transfer.EventTransferStarted, transfer.EventTransferFinished:
			logger.Info(fmt.Sprintf("[Event] %s", event.String()), "msg", msg)
		default:
			logger.Debug(fmt.Sprintf("[Event] %s", event.String()), "msg", msg)
		}
	})

	router := ivy.NewRouter()

	router.Post("/send/{connectionID}", func(c *ivy.Context) error {
		connectionID := c.PathParam("connectionID")

		wait := c.QueryParam("wait")
		if wait == "" {
			wait = "30s"
		}

		waitDuration, err := time.ParseDuration(wait)
		if err != nil {
			return fmt.Errorf("bad wait time, must be a valid go time.Duration")
		}

		if waitDuration.Seconds() > maxWaitDuration.Seconds() {
			return fmt.Errorf("invalid wait duration, must be <= %s", *maxWait)
		}

		logger.Info("sender active", "connectionID", connectionID, "wait", waitDuration)

		if err := Send(c, SendParams{
			TransferManager: tm,
			ConnectionID:    connectionID,
			MaxWaitDuration: waitDuration,
		}); err != nil {
			return err
		}

		return nil
	})

	router.Get("/receive/{connectionID}", func(c *ivy.Context) error {
		connectionID := c.PathParam("connectionID")
		logger.Debug("receiver active", "connectionID", connectionID)

		return tm.StartTransfer(connectionID, c)
	})

	if err := http.ListenAndServe(*addr, router); err != nil {
		logger.Error(err, "starting http server", "addr", addr)
	}
}
