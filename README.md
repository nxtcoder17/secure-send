# Secure Send

A lightweight, secure file transfer service written in Go that enables direct client-to-client file transfers over HTTPs

## Features

- Fast and efficient file transfers
- Real-time transfer progress monitoring
- Configurable wait times for receivers
- Event-based transfer tracking
- Cross-platform support (Linux/ARM64, AMD64)
- Containerized deployment ready

### Usage

### Sender (Client 1)
```bash
curl -X POST <Host>/send/{connectionID} --data-binary @file.txt
```

### Receiver (Client 2)
```bash
curl <Host>/receive/{connectionID} > /tmp/file.txt
```
