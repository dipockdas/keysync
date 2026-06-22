package wormhole

import (
	"context"
	"io"

	william "github.com/psanford/wormhole-william/wormhole"
)

type williamClient struct {
	client *william.Client
}

func New() Transport {
	return newWithClient(&williamClient{client: &william.Client{}}, SessionTimeout)
}

func (c *williamClient) SendFile(ctx context.Context, filename string, reader io.ReadSeeker) (string, <-chan protocolSendResult, error) {
	code, status, err := c.client.SendFile(ctx, filename, reader)
	if err != nil {
		return "", nil, err
	}
	result := make(chan protocolSendResult, 1)
	go func() {
		defer close(result)
		select {
		case williamResult, ok := <-status:
			if !ok {
				result <- protocolSendResult{Err: io.ErrUnexpectedEOF}
				return
			}
			result <- protocolSendResult{OK: williamResult.OK, Err: williamResult.Error}
		case <-ctx.Done():
			result <- protocolSendResult{Err: ctx.Err()}
		}
	}()
	return code, result, nil
}

func (c *williamClient) Receive(ctx context.Context, code string) (protocolIncoming, error) {
	incoming, err := c.client.Receive(ctx, code)
	if err != nil {
		return protocolIncoming{}, err
	}
	transferType := protocolTransferText
	switch incoming.Type {
	case william.TransferFile:
		transferType = protocolTransferFile
	case william.TransferDirectory:
		transferType = protocolTransferDirectory
	}
	return protocolIncoming{
		Name:   incoming.Name,
		Type:   transferType,
		Size:   incoming.TransferBytes64,
		Body:   incoming,
		Reject: incoming.Reject,
	}, nil
}
