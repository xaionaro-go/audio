package types

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
)

type PlayerPCM interface {
	io.Closer
	Ping(context.Context) error
	PlayPCM(
		ctx context.Context,
		sampleRate SampleRate,
		channels Channel,
		format PCMFormat,
		bufferSize time.Duration,
		reader io.Reader,
	) (PlayStream, error)
}

type PCMFormat uint

const (
	PCMFormatUndefined = PCMFormat(iota)
	PCMFormatU8
	PCMFormatS16LE
	PCMFormatS16BE
	PCMFormatFloat32LE
	PCMFormatFloat32BE
	PCMFormatS24LE
	PCMFormatS24BE
	PCMFormatS32LE
	PCMFormatS32BE
	PCMFormatFloat64LE
	PCMFormatFloat64BE
	PCMFormatS64LE
	PCMFormatS64BE
	EndOfPCMFormat
)

func (f PCMFormat) Size() uint32 {
	switch f {
	case PCMFormatUndefined:
		return math.MaxUint32
	case PCMFormatU8:
		return 1
	case PCMFormatS16LE, PCMFormatS16BE:
		return 2
	case PCMFormatS24LE, PCMFormatS24BE:
		return 3
	case PCMFormatFloat32LE, PCMFormatFloat32BE, PCMFormatS32LE, PCMFormatS32BE:
		return 4
	case PCMFormatFloat64LE, PCMFormatFloat64BE, PCMFormatS64LE, PCMFormatS64BE:
		return 8
	default:
		return math.MaxUint32
	}
}

func (f PCMFormat) String() string {
	switch f {
	case PCMFormatUndefined:
		return "<undefined>"
	case PCMFormatS16LE:
		return "s16le"
	case PCMFormatFloat32LE:
		return "f32le"
	default:
		return fmt.Sprintf("<unexpected_value_%d>", f)
	}
}

func PCMFormatFromString(in string) PCMFormat {
	in = strings.ToLower(in)
	for fmt := PCMFormatUndefined + 1; fmt < EndOfPCMFormat; fmt++ {
		if strings.ToLower(fmt.String()) == in {
			return fmt
		}
	}
	return PCMFormatUndefined
}

type SampleRate uint32

type Channel uint32

type Encoding interface {
	BytesPerSample() uint
	BytesForSecond() uint
	BytesForDuration(d time.Duration) uint64
}

type EncodingPCM struct {
	PCMFormat
	SampleRate
}

func (pcm EncodingPCM) BytesPerSample() uint {
	return uint(pcm.PCMFormat.Size())
}

func (pcm EncodingPCM) BytesForSecond() uint {
	return uint(pcm.BytesPerSample()) * uint(pcm.SampleRate)
}

func (pcm EncodingPCM) BytesForDuration(d time.Duration) uint64 {
	return (uint64(pcm.SampleRate) * uint64(d.Microseconds()) / 1000000) * uint64(pcm.BytesPerSample())
}
