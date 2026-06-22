package wormhole

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestSendReturnsCodeBeforeTransferCompletion(t *testing.T) {
	status := make(chan protocolSendResult)
	client := &fakeProtocolClient{
		sendCode:   "7-purple-dolphin",
		sendStatus: status,
	}
	transport := newWithClient(client, time.Second)

	code, result, err := transport.Send(context.Background(), "example.keysync.ksx", []byte("synthetic-encrypted-bundle"))
	if err != nil {
		t.Fatal(err)
	}
	if code != "7-purple-dolphin" {
		t.Fatalf("code = %q", code)
	}
	select {
	case <-result:
		t.Fatal("result completed before protocol status")
	default:
	}
	status <- protocolSendResult{OK: true}
	got := <-result
	if !got.OK || got.Err != nil {
		t.Fatalf("result = %#v", got)
	}
	if client.sentName != "example.keysync.ksx" || string(client.sentData) != "synthetic-encrypted-bundle" {
		t.Fatalf("sent = %q %q", client.sentName, client.sentData)
	}
}

func TestSendTimesOutAndDoesNotExposePayloadInError(t *testing.T) {
	client := &fakeProtocolClient{
		sendCode:   "7-purple-dolphin",
		sendStatus: make(chan protocolSendResult),
	}
	transport := newWithClient(client, 10*time.Millisecond)
	_, result, err := transport.Send(context.Background(), "example.keysync.ksx", []byte("synthetic-sensitive-ciphertext"))
	if err != nil {
		t.Fatal(err)
	}
	got := <-result
	if !errors.Is(got.Err, ErrTimeout) {
		t.Fatalf("error = %v, want ErrTimeout", got.Err)
	}
	if strings.Contains(got.Err.Error(), "synthetic-sensitive-ciphertext") {
		t.Fatalf("error exposes payload: %v", got.Err)
	}
}

func TestReceiveReturnsBoundedFileBytes(t *testing.T) {
	client := &fakeProtocolClient{incoming: protocolIncoming{
		Name: "example.keysync.ksx",
		Type: protocolTransferFile,
		Size: int64(len("synthetic-encrypted-bundle")),
		Body: strings.NewReader("synthetic-encrypted-bundle"),
	}}
	transport := newWithClient(client, time.Second)

	filename, data, err := transport.Receive(context.Background(), "7-purple-dolphin")
	if err != nil {
		t.Fatal(err)
	}
	if filename != "example.keysync.ksx" || string(data) != "synthetic-encrypted-bundle" {
		t.Fatalf("received = %q %q", filename, data)
	}
}

func TestReceiveTimesOut(t *testing.T) {
	client := &fakeProtocolClient{blockReceive: true}
	transport := newWithClient(client, 10*time.Millisecond)
	_, _, err := transport.Receive(context.Background(), "7-purple-dolphin")
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("error = %v, want ErrTimeout", err)
	}
}

func TestParentCancellationIsInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &fakeProtocolClient{blockReceive: true}
	transport := newWithClient(client, time.Second)
	_, _, err := transport.Receive(ctx, "7-purple-dolphin")
	if !errors.Is(err, ErrInterrupted) {
		t.Fatalf("error = %v, want ErrInterrupted", err)
	}
}

func TestReceiveRejectsInvalidCodeTypeNameAndSize(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		incoming protocolIncoming
		want     error
		rejected bool
	}{
		{name: "invalid code", code: "not-a-code", want: ErrInvalidCode},
		{name: "wrong type", code: "7-purple-dolphin", incoming: protocolIncoming{Name: "x", Type: protocolTransferText, Size: 1, Body: strings.NewReader("x")}, want: ErrInvalidPayload, rejected: true},
		{name: "unsafe name", code: "7-purple-dolphin", incoming: protocolIncoming{Name: "../x.keysync.ksx", Type: protocolTransferFile, Size: 1, Body: strings.NewReader("x")}, want: ErrInvalidPayload, rejected: true},
		{name: "claimed too large", code: "7-purple-dolphin", incoming: protocolIncoming{Name: "x.keysync.ksx", Type: protocolTransferFile, Size: MaxPayloadSize + 1, Body: strings.NewReader("x")}, want: ErrInvalidPayload, rejected: true},
		{name: "actual too large", code: "7-purple-dolphin", incoming: protocolIncoming{Name: "x.keysync.ksx", Type: protocolTransferFile, Size: 1, Body: bytes.NewReader(bytes.Repeat([]byte("x"), int(MaxPayloadSize+1)))}, want: ErrInvalidPayload},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeProtocolClient{incoming: tt.incoming}
			transport := newWithClient(client, time.Second)
			_, _, err := transport.Receive(context.Background(), tt.code)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if tt.rejected && !client.rejected {
				t.Fatal("invalid incoming transfer was not rejected")
			}
		})
	}
}

func TestTransportClassifiesProtocolFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "rejected", err: errors.New("receiver rejected transfer"), want: ErrRejected},
		{name: "unavailable", err: errors.New("websocket dial failed"), want: ErrUnavailable},
		{name: "interrupted", err: io.ErrUnexpectedEOF, want: ErrInterrupted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := make(chan protocolSendResult, 1)
			status <- protocolSendResult{Err: tt.err}
			transport := newWithClient(&fakeProtocolClient{sendCode: "7-purple-dolphin", sendStatus: status}, time.Second)
			_, result, err := transport.Send(context.Background(), "x.keysync.ksx", []byte("synthetic-payload"))
			if err != nil {
				t.Fatal(err)
			}
			if got := (<-result).Err; !errors.Is(got, tt.want) {
				t.Fatalf("error = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewUsesWilliamDefaultsAndFiveMinuteTimeout(t *testing.T) {
	transport, ok := New().(*Adapter)
	if !ok {
		t.Fatalf("New() type = %T", New())
	}
	if transport.timeout != SessionTimeout || SessionTimeout != 5*time.Minute {
		t.Fatalf("timeout = %s", transport.timeout)
	}
	client, ok := transport.client.(*williamClient)
	if !ok {
		t.Fatalf("client type = %T", transport.client)
	}
	if client.client.RendezvousURL != "" || client.client.TransitRelayAddress != "" {
		t.Fatal("New() overrides wormhole-william default infrastructure")
	}
}

type fakeProtocolClient struct {
	sendCode     string
	sendStatus   <-chan protocolSendResult
	sendErr      error
	sentName     string
	sentData     []byte
	incoming     protocolIncoming
	receiveErr   error
	rejected     bool
	blockReceive bool
}

func (c *fakeProtocolClient) SendFile(_ context.Context, filename string, reader io.ReadSeeker) (string, <-chan protocolSendResult, error) {
	c.sentName = filename
	c.sentData, _ = io.ReadAll(reader)
	return c.sendCode, c.sendStatus, c.sendErr
}

func (c *fakeProtocolClient) Receive(ctx context.Context, _ string) (protocolIncoming, error) {
	if c.blockReceive {
		<-ctx.Done()
		return protocolIncoming{}, ctx.Err()
	}
	incoming := c.incoming
	incoming.Reject = func() error {
		c.rejected = true
		return nil
	}
	return incoming, c.receiveErr
}
