package types

import (
	"context"
	"io"
)

type RecorderPCM interface {
	io.Closer
	Ping(context.Context) error
	RecordPCM(
		ctx context.Context,
		sampleRate SampleRate,
		channels Channel,
		format PCMFormat,
		writer io.Writer,
	) (RecordStream, error)
}
