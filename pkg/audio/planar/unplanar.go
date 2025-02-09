package planar

import (
	"fmt"
	"io"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type UnplanarReader struct {
	Backend    io.Reader
	Channels   audio.Channel
	SampleSize uint
	Buffer     []byte
}

var _ io.Reader = (*UnplanarReader)(nil)

func NewUnplanarReader(
	backend io.Reader,
	channels audio.Channel,
	sampleSize uint,
	bufferSize uint,
) *UnplanarReader {
	if bufferSize%(sampleSize*uint(channels)) != 0 {
		panic(fmt.Errorf("buffer size in not a multiple of sampleSize*channels: %d %% %d*%d != 0", bufferSize, sampleSize, uint(channels)))
	}
	return &UnplanarReader{
		Backend:    backend,
		Channels:   channels,
		SampleSize: sampleSize,
		Buffer:     make([]byte, bufferSize),
	}
}

func (r *UnplanarReader) Read(p []byte) (int, error) {
	shortestMessageSize := int(r.Channels) * int(r.SampleSize)
	if len(p) < shortestMessageSize {
		return 0, fmt.Errorf("the provided output buffer is too short: %d < %d", len(p), shortestMessageSize)
	}
	requestLength := (len(p) / shortestMessageSize) * shortestMessageSize
	if requestLength > len(r.Buffer) {
		requestLength = len(r.Buffer)
	}

	n, err := r.Backend.Read(r.Buffer[:requestLength])
	if err != nil {
		return n, fmt.Errorf("unable to read from the backend: %w", err)
	}
	buf := r.Buffer[:n]
	if len(buf)%shortestMessageSize != 0 {
		return n, fmt.Errorf("expected a message length that is a multiple of %d, but received %d", shortestMessageSize, len(buf))
	}

	samplesPerChan := len(buf) / int(r.Channels) / int(r.SampleSize)

	for ch := audio.Channel(0); ch < r.Channels; ch++ {
		inIdxOffset := int(ch) * samplesPerChan * int(r.SampleSize)
		outIdxOffset := int(r.SampleSize) * int(ch)
		for samplePos := 0; samplePos < samplesPerChan; samplePos++ {
			inIdxOffset := inIdxOffset + samplePos*int(r.SampleSize)
			outIdxOffset := outIdxOffset + samplePos*(int(r.SampleSize)*int(r.Channels))
			for sampleByte := 0; sampleByte < int(r.SampleSize); sampleByte++ {
				inIdx := inIdxOffset + sampleByte
				outIdx := outIdxOffset + sampleByte
				p[outIdx] = buf[inIdx]
			}
		}
	}

	return n, nil
}
