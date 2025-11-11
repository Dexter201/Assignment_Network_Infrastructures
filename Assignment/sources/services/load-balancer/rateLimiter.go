package main

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

// rateLimitedReader wraps an io.Reader and uses a rate.Limiter to throttle bytes read.
type rateLimitedReader struct {
	reader  io.Reader
	limiter *rate.Limiter
}

func createRateLimitedReader(reader io.Reader, limiter *rate.Limiter) io.Reader {
	return &rateLimitedReader{reader: reader, limiter: limiter}
}

// we have to implempent the Read method to fullfil the Readers interface
func (reader *rateLimitedReader) Read(p []byte) (int, error) {
	n, err := reader.reader.Read(p)
	if n > 0 {
		_ = reader.limiter.WaitN(context.Background(), n)
	}
	return n, err
}
