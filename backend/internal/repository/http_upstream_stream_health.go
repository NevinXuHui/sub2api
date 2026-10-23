package repository

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// This observer never retries or replays a body. It only changes the transport
// selected by later requests and closes idle connections, never active streams.
func (s *httpUpstreamService) observeOpenAIHTTP2Body(ctx context.Context, profile service.HTTPUpstreamProfile, entry *upstreamClientEntry, body io.ReadCloser) io.ReadCloser {
	if body == nil || profile != service.HTTPUpstreamProfileOpenAI || entry.protocolMode != upstreamProtocolModeOpenAIH2 {
		return body
	}
	return &upstreamHealthBody{ReadCloser: body, observe: func(err error) {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		if errors.Is(err, io.EOF) {
			s.recordOpenAIHTTP2Success(profile, entry.protocolMode, entry.proxyKey)
		} else if isOpenAIHTTP2CompatibilityError(err) {
			s.recordOpenAIHTTP2Failure(profile, entry.protocolMode, entry.proxyKey, err)
			entry.client.CloseIdleConnections()
		}
	}}
}

type upstreamHealthBody struct {
	io.ReadCloser
	once    sync.Once
	observe func(error)
}

func (b *upstreamHealthBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil {
		b.once.Do(func() { b.observe(err) })
	}
	return n, err
}
