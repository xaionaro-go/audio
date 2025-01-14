package audio

import (
	"context"
	"io"
	"time"
)

type PlayerPCMDummy struct{}

var _ PlayerPCM = PlayerPCMDummy{}

func (PlayerPCMDummy) Close() error {
	return nil
}

func (PlayerPCMDummy) Ping(context.Context) error {
	return nil
}

func (PlayerPCMDummy) PlayPCM(
	ctx context.Context,
	sampleRate SampleRate,
	channels Channel,
	format PCMFormat,
	bufferSize time.Duration,
	reader io.Reader,
) (PlayStream, error) {
	return StreamDummy{}, nil
}

type StreamDummy struct{}

var _ Stream = StreamDummy{}

func (StreamDummy) Drain() error {
	return nil
}

func (StreamDummy) Close() error {
	return nil
}
