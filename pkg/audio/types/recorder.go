package types

import (
	"io"
)

type RecorderPCM interface {
	Ping() error
	RecordPCM(
		sampleRate SampleRate,
		channels Channel,
		format PCMFormat,
		writer io.Writer,
	) (RecordStream, error)
}
