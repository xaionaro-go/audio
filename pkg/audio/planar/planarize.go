package planar

import (
	"fmt"
	"io"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type PlanarizeReader struct {
	Backend    io.Reader
	Channels   audio.Channel
	SampleSize uint
	Buffer     []byte
}

var _ io.Reader = (*PlanarizeReader)(nil)

func NewPlanarizeReader(
	backend io.Reader,
	channels audio.Channel,
	sampleSize uint,
	bufferSize uint,
) *PlanarizeReader {
	if bufferSize%(sampleSize*uint(channels)) != 0 {
		panic(fmt.Errorf("buffer size in not a multiple of sampleSize*channels: %d %% %d*%d != 0", bufferSize, sampleSize, uint(channels)))
	}
	return &PlanarizeReader{
		Backend:    backend,
		Channels:   channels,
		SampleSize: sampleSize,
		Buffer:     make([]byte, bufferSize),
	}
}

func (r *PlanarizeReader) Read(p []byte) (int, error) {
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
	if n > requestLength {
		return n, fmt.Errorf("received more bytes than requested: %d > %d", n, requestLength)
	}

	err = Planarize(r.Channels, r.SampleSize, p[:n], r.Buffer[:n])
	if err != nil {
		return n, fmt.Errorf("unable to planarize: %w", err)
	}

	return n, nil
}

func Planarize(channels audio.Channel, sampleSize uint, output, input []byte) error {
	shortestMessageSize := int(channels) * int(sampleSize)
	if len(input) < shortestMessageSize {
		return fmt.Errorf("the provided input buffer is too short: %d < %d", len(input), shortestMessageSize)
	}
	if len(input)%shortestMessageSize != 0 {
		return fmt.Errorf("expected a message length that is a multiple of %d, but received %d", shortestMessageSize, len(input))
	}
	if len(input) != len(output) {
		return fmt.Errorf("the lengths of input and output are not equal: %d != %d", len(input), len(output))
	}

	samplesPerChan := len(input) / int(channels) / int(sampleSize)

	for ch := audio.Channel(0); ch < channels; ch++ {
		inIdxOffset := int(sampleSize) * int(ch)
		outIdxOffset := int(ch) * samplesPerChan * int(sampleSize)
		for samplePos := 0; samplePos < samplesPerChan; samplePos++ {
			inIdxOffset2 := inIdxOffset + samplePos*(int(sampleSize)*int(channels))
			outIdxOffset2 := outIdxOffset + samplePos*int(sampleSize)
			for sampleByte := 0; sampleByte < int(sampleSize); sampleByte++ {
				inIdx := inIdxOffset2 + sampleByte
				outIdx := outIdxOffset2 + sampleByte
				output[outIdx] = input[inIdx]
			}
		}
	}

	return nil
}
