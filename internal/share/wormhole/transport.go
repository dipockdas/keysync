package wormhole

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dipockdas/keysync/internal/share/ksx"
)

const (
	SessionTimeout = 5 * time.Minute
	MaxPayloadSize = int64(ksx.MaxBundleSize)
)

var (
	ErrInvalidCode    = errors.New("invalid Wormhole code")
	ErrRejected       = errors.New("Wormhole transfer rejected")
	ErrInterrupted    = errors.New("Wormhole transfer interrupted")
	ErrTimeout        = errors.New("Wormhole transfer timed out")
	ErrUnavailable    = errors.New("Wormhole service unavailable")
	ErrInvalidPayload = errors.New("invalid Wormhole payload")
)

type Result struct {
	OK  bool
	Err error
}

type Transport interface {
	Send(ctx context.Context, filename string, data []byte) (code string, result <-chan Result, err error)
	Receive(ctx context.Context, code string) (filename string, data []byte, err error)
}

type Adapter struct {
	client  protocolClient
	timeout time.Duration
}

type protocolSendResult struct {
	OK  bool
	Err error
}

type protocolTransferType uint8

const (
	protocolTransferText protocolTransferType = iota
	protocolTransferFile
	protocolTransferDirectory
)

type protocolIncoming struct {
	Name   string
	Type   protocolTransferType
	Size   int64
	Body   io.Reader
	Reject func() error
}

type protocolClient interface {
	SendFile(ctx context.Context, filename string, reader io.ReadSeeker) (string, <-chan protocolSendResult, error)
	Receive(ctx context.Context, code string) (protocolIncoming, error)
}

func newWithClient(client protocolClient, timeout time.Duration) *Adapter {
	return &Adapter{client: client, timeout: timeout}
}

func (a *Adapter) Send(ctx context.Context, filename string, data []byte) (string, <-chan Result, error) {
	if err := validateFilename(filename); err != nil {
		return "", nil, err
	}
	if len(data) == 0 || int64(len(data)) > MaxPayloadSize {
		return "", nil, fmt.Errorf("%w: encrypted bundle size is invalid", ErrInvalidPayload)
	}

	sessionCtx, cancel := context.WithTimeout(ctx, a.timeout)
	code, protocolResult, err := a.client.SendFile(sessionCtx, filename, bytes.NewReader(data))
	if err != nil {
		cancel()
		return "", nil, classifyError(sessionCtx, err)
	}
	if err := validateCode(code); err != nil {
		cancel()
		return "", nil, err
	}

	result := make(chan Result, 1)
	go func() {
		defer cancel()
		defer close(result)
		select {
		case status, ok := <-protocolResult:
			if !ok {
				result <- Result{Err: ErrInterrupted}
				return
			}
			if status.Err != nil {
				result <- Result{Err: classifyError(sessionCtx, status.Err)}
				return
			}
			if !status.OK {
				result <- Result{Err: ErrInterrupted}
				return
			}
			result <- Result{OK: true}
		case <-sessionCtx.Done():
			result <- Result{Err: classifyError(sessionCtx, sessionCtx.Err())}
		}
	}()
	return code, result, nil
}

func (a *Adapter) Receive(ctx context.Context, code string) (string, []byte, error) {
	if err := validateCode(code); err != nil {
		return "", nil, err
	}
	sessionCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	incoming, err := a.client.Receive(sessionCtx, code)
	if err != nil {
		return "", nil, classifyError(sessionCtx, err)
	}
	reject := func() {
		if incoming.Reject != nil {
			_ = incoming.Reject()
		}
	}
	if incoming.Type != protocolTransferFile {
		reject()
		return "", nil, fmt.Errorf("%w: expected an encrypted file", ErrInvalidPayload)
	}
	if err := validateFilename(incoming.Name); err != nil {
		reject()
		return "", nil, err
	}
	if incoming.Size <= 0 || incoming.Size > MaxPayloadSize {
		reject()
		return "", nil, fmt.Errorf("%w: offered size is invalid", ErrInvalidPayload)
	}
	if incoming.Body == nil {
		reject()
		return "", nil, fmt.Errorf("%w: transfer has no body", ErrInvalidPayload)
	}
	data, err := io.ReadAll(io.LimitReader(incoming.Body, MaxPayloadSize+1))
	if err != nil {
		return "", nil, classifyError(sessionCtx, err)
	}
	if len(data) == 0 || int64(len(data)) > MaxPayloadSize || int64(len(data)) != incoming.Size {
		return "", nil, fmt.Errorf("%w: received size does not match offer", ErrInvalidPayload)
	}
	return incoming.Name, data, nil
}

func validateCode(code string) error {
	parts := strings.SplitN(code, "-", 2)
	if len(parts) != 2 || parts[1] == "" {
		return ErrInvalidCode
	}
	nameplate, err := strconv.Atoi(parts[0])
	if err != nil || nameplate <= 0 {
		return ErrInvalidCode
	}
	return nil
}

func validateFilename(filename string) error {
	if filename == "" || len(filename) > 255 || strings.ContainsRune(filename, '\x00') || filepath.Base(filename) != filename || filename == "." || filename == ".." {
		return fmt.Errorf("%w: unsafe filename", ErrInvalidPayload)
	}
	return nil
}

func classifyError(ctx context.Context, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ErrTimeout
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return ErrInterrupted
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "reject"), strings.Contains(message, "declin"):
		return ErrRejected
	case strings.Contains(message, "dial"), strings.Contains(message, "websocket"), strings.Contains(message, "connect"), strings.Contains(message, "unavailable"), strings.Contains(message, "network"):
		return ErrUnavailable
	case strings.Contains(message, "nameplate"), strings.Contains(message, "invalid code"):
		return ErrInvalidCode
	default:
		return ErrInterrupted
	}
}
