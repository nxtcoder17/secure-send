package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/nxtcoder17/fastlog"
	"github.com/nxtcoder17/ivy"
	"github.com/nxtcoder17/secure-send/pkg/transfer"
)

var (
	BuiltAt string
	Version string
)

func init() {
	if BuiltAt == "" {
		BuiltAt = time.Now().Format(time.RFC822)
	}

	if Version == "" {
		Version = "aurora"
	}
}

func main() {
	addr := flag.String("addr", ":3000", "--addr [host]:<port>")
	maxWait := flag.String("max-wait", "120s", "--max-wait <duration>")
	debug := flag.Bool("debug", false, "--debug")
	flag.Parse()

	maxWaitDuration, err := time.ParseDuration(*maxWait)
	if err != nil {
		panic(err)
	}

	logger := fastlog.New(fastlog.Options{
		Writer:        os.Stderr,
		Format:        fastlog.ConsoleFormat,
		ShowDebugLogs: *debug,
		ShowCaller:    true,
		EnableColors:  true,
	})

	slog.SetDefault(logger.Slog())

	fmt.Printf(`
 ▗▄▄▖▗▄▄▄▖ ▗▄▄▖▗▖ ▗▖▗▄▄▖ ▗▄▄▄▖   ▗▄▄▖▗▄▄▄▖▗▖  ▗▖▗▄▄▄ 
▐▌   ▐▌   ▐▌   ▐▌ ▐▌▐▌ ▐▌▐▌     ▐▌   ▐▌   ▐▛▚▖▐▌▐▌  █
 ▝▀▚▖▐▛▀▀▘▐▌   ▐▌ ▐▌▐▛▀▚▖▐▛▀▀▘   ▝▀▚▖▐▛▀▀▘▐▌ ▝▜▌▐▌  █
▗▄▄▞▘▐▙▄▄▖▝▚▄▄▖▝▚▄▞▘▐▌ ▐▌▐▙▄▄▖  ▗▄▄▞▘▐▙▄▄▖▐▌  ▐▌▐▙▄▄▀

› Version %s
› Built on %s
› Http Server running on %s
`, Version, BuiltAt, *addr)

	tm := transfer.NewInMemoryTransferManager()

	tm.Subscribe(func(event transfer.Event, msg string, kv ...any) {
		kv = append(kv, "event", event.String())
		switch event {
		case transfer.EventTransferStarted, transfer.EventTransferFinished:
			logger.Info(msg, kv...)
		default:
			logger.Debug(msg, kv...)
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

		logger.Info("sender active", "connectionID", connectionID, "wait", fmt.Sprintf("%ds", int64(waitDuration.Seconds())))

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
		logger.Info("receiver active", "connectionID", connectionID)

		return tm.StartTransfer(connectionID, c)
	})

	if err := http.ListenAndServe(*addr, router); err != nil {
		logger.Error("failed to start http server", "addr", addr, "err", err)
		os.Exit(1)
	}
}
