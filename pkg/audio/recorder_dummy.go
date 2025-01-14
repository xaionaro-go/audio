package audio

import (
	"context"
	"io"
)

type RecorderPCMDummy struct{}

var _ RecorderPCM = RecorderPCMDummy{}

func (RecorderPCMDummy) Close() error {
	return nil
}

func (RecorderPCMDummy) Ping(context.Context) error {
	return nil
}

func (RecorderPCMDummy) RecordPCM(
	ctx context.Context,
	sampleRate SampleRate,
	channels Channel,
	format PCMFormat,
	writer io.Writer,
) (RecordStream, error) {
	return StreamDummy{}, nil
}
