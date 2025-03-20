package main

import (
	"bufio"
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

type Distributor struct {
	ch chan []byte

	debug bool

	debugSender   io.Writer
	debugReceiver io.Writer

	mu sync.Mutex
}

var (
	BuiltAt string

	// to be set by runtime flags
	debug bool
)

func (d *Distributor) Send(b []byte) {
	// INFO: this data must be copied to another []byte first
	// [go io.Writer](https://pkg.go.dev/io#Writer)
	// [stack overflow](https://stackoverflow.com/a/41250437)
	t := make([]byte, len(b))
	copy(t, b)

	if d.debug {
		fmt.Fprintf(d.debugSender, "size: %d, md5: %x\n", len(t), md5.Sum(t))
	}
	d.ch <- t
}

func (d *Distributor) Close() {
	close(d.ch)
}

func (d *Distributor) Receive(callback func(b []byte) error) {
	for b := range d.ch {
		if d.debug {
			fmt.Fprintf(d.debugReceiver, "size: %d, md5: %x\n", len(b), md5.Sum(b))
		}
		if err := callback(b); err != nil {
			return
		}
	}
}

func NewDistributor() *Distributor {
	d := &Distributor{
		ch: make(chan []byte, 1),
		mu: sync.Mutex{},
	}

	if debug {
		d.debug = true

		var err error
		d.debugSender, err = os.Create("_send_debug.log")
		if err != nil {
			panic(err)
		}
		d.debugReceiver, err = os.Create("_receive_debug.log")
		if err != nil {
			panic(err)
		}
	}

	return d
}

func main() {
	var addr string
	flag.StringVar(&addr, "addr", ":7890", "--addr [host:]port")
	flag.BoolVar(&debug, "debug", false, "--debug")
	flag.Parse()

	mux := http.NewServeMux()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level: func() slog.Level {
			if debug {
				return slog.LevelDebug
			}
			return slog.LevelInfo
		}(),
	}))

	clients := make(map[string]*Distributor)

	mu := sync.Mutex{}

	mux.HandleFunc("POST /send/{token}", func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")

		wait := r.URL.Query().Get("wait")
		if wait == "" {
			wait = "10s"
		}

		waitDuration, err := time.ParseDuration(wait)
		if err != nil {
			http.Error(w, "bad wait time, must be a valid go time.Duration", http.StatusBadRequest)
			return
		}

		if f := waitDuration.Seconds(); f > 120 {
			http.Error(w, "invalid wait duration, must be < 120 seconds", http.StatusBadRequest)
			return
		}

		logger.Debug("sender active", "token", token)

		var reader io.Reader

		switch r.URL.Query().Get("type") {
		case "form":
			{
				logger.Info("parsing multipart form")
				start := time.Now()

				// r.Body = http.MaxBytesReader(w, r.Body, 32<<20+512)
				if err := r.ParseMultipartForm(32 << 20); err != nil {
					logger.Error("unable to parse form, got", "err", err)
					http.Error(w, "Unable to parse form", http.StatusBadRequest)
					return
				}
				logger.Info("parsed multipart form", "took", fmt.Sprintf("%.2fs", time.Since(start).Seconds()))

				// Retrieve the file from the form
				file, fh, err := r.FormFile("file")
				if err != nil {
					http.Error(w, "Error retrieving the file", http.StatusBadRequest)
					return
				}
				defer file.Close()
				logger.Debug("file header", "fh", fh)
				reader = file
			}
		default:
			defer r.Body.Close()
			reader = bufio.NewReader(r.Body)
		}

		waitTill := time.Now().Add(waitDuration)
		for time.Since(waitTill) < 0 {
			_, ok := clients[token]
			if ok {
				break
			}

			fmt.Fprintf(w, "\rIDLE connection, no receiver found, waiting another %.2fs\n", time.Until(waitTill).Seconds())
			w.(http.Flusher).Flush()
			<-time.After(200 * time.Millisecond)
		}

		msg := make([]byte, 0xffff)

		transferred := 0

		for {
			n, err := reader.Read(msg)
			if err != nil {
				if errors.Is(err, io.EOF) {
					if ch, ok := clients[token]; ok {
						ch.Send(msg[:n])
						ch.Close()
						break
					}
					logger.Error("no token found")
				}
				logger.Error("while reading file, got", "err", err)
				return
			}

			ch, ok := clients[token]
			if !ok {
				return
			}

			ch.Send(msg[:n])
			transferred += n

			fmt.Fprintf(w, "\rwritten %.2f MBs (%d bytes)", float64(transferred)/1024/1024, transferred)
			w.(http.Flusher).Flush()
		}
		fmt.Fprintf(w, "\rwritten %.2f MBs (%d bytes)", float64(transferred)/1024/1024, transferred)
	})

	mux.HandleFunc("GET /receive/{token}", func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")

		ch := NewDistributor()
		mu.Lock()
		clients[token] = ch
		mu.Unlock()

		defer func() {
			mu.Lock()
			delete(clients, token)
			mu.Unlock()
		}()

		logger.Info("client waiting", "token", token)

		logger.Debug("waiting to read")
		w.Header().Set("Content-Type", "application/octet-stream")

		received := 0
		written := 0

		ch.Receive(func(b []byte) error {
			received += len(b)
			n, err := w.Write(b)
			if err != nil {
				return err
			}
			w.(http.Flusher).Flush()
			written += n
			return nil
		})

		logger.Debug("finished", "received", received, "written", written)
	})

	mux.HandleFunc("GET /_healthy", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	if BuiltAt == "" {
		BuiltAt = time.Now().Format("11:30 AM Sunday, Nov 11, 2023")
	}

	fmt.Printf(`
 ▗▄▄▖▗▄▄▄▖ ▗▄▄▖▗▖ ▗▖▗▄▄▖ ▗▄▄▄▖     ▗▄▄▖▗▄▄▄▖▗▖  ▗▖▗▄▄▄ 
▐▌   ▐▌   ▐▌   ▐▌ ▐▌▐▌ ▐▌▐▌       ▐▌   ▐▌   ▐▛▚▖▐▌▐▌  █
 ▝▀▚▖▐▛▀▀▘▐▌   ▐▌ ▐▌▐▛▀▚▖▐▛▀▀▘     ▝▀▚▖▐▛▀▀▘▐▌ ▝▜▌▐▌  █
▗▄▄▞▘▐▙▄▄▖▝▚▄▄▖▝▚▄▞▘▐▌ ▐▌▐▙▄▄▖    ▗▄▄▞▘▐▙▄▄▖▐▌  ▐▌▐▙▄▄▀

Built at %s
`, BuiltAt)

	logger.Info("HTTP server starting", "at", addr)
	http.ListenAndServe(addr, mux)
}
