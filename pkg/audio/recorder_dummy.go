package audio

import (
	"io"
)

type RecorderPCMDummy struct{}

var _ RecorderPCM = RecorderPCMDummy{}

func (RecorderPCMDummy) Ping() error {
	return nil
}

func (RecorderPCMDummy) RecordPCM(
	sampleRate SampleRate,
	channels Channel,
	format PCMFormat,
	writer io.Writer,
) (RecordStream, error) {
	return StreamDummy{}, nil
}
