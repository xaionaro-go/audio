package resampler

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

const (
	distanceStep = 10000
)

type Format struct {
	Channels   audio.Channel
	SampleRate audio.SampleRate
	PCMFormat  types.PCMFormat
}

type precalculated struct {
	inSampleSize    uint
	outSampleSize   uint
	inNumAvg        uint
	outNumRepeat    uint
	outDistanceStep uint64
}

type Resampler struct {
	inReader    io.Reader
	inFormat    Format
	outFormat   Format
	inDistance  uint64
	outDistance uint64
	locker      sync.Mutex
	buffer      []byte
	precalculated
}

func getFloat64(f types.PCMFormat, p []byte) float64 {
	switch f {
	case types.PCMFormatU8:
		return (float64(p[0]) - 128) / 128
	case types.PCMFormatS16LE:
		return float64(int16(binary.LittleEndian.Uint16(p))) / 32768
	case types.PCMFormatS16BE:
		return float64(int16(binary.BigEndian.Uint16(p))) / 32768
	case types.PCMFormatS24LE:
		val := int32(uint32(p[0]) | uint32(p[1])<<8 | uint32(p[2])<<16)
		if val&0x800000 != 0 {
			val |= -16777216
		}
		return float64(val) / 8388608
	case types.PCMFormatS24BE:
		val := int32(uint32(p[2]) | uint32(p[1])<<8 | uint32(p[0])<<16)
		if val&0x800000 != 0 {
			val |= -16777216
		}
		return float64(val) / 8388608
	case types.PCMFormatS32LE:
		return float64(int32(binary.LittleEndian.Uint32(p))) / 2147483648
	case types.PCMFormatS32BE:
		return float64(int32(binary.BigEndian.Uint32(p))) / 2147483648
	case types.PCMFormatS64LE:
		return float64(int64(binary.LittleEndian.Uint64(p))) / 9223372036854775808
	case types.PCMFormatS64BE:
		return float64(int64(binary.BigEndian.Uint64(p))) / 9223372036854775808
	case types.PCMFormatFloat32LE:
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(p)))
	case types.PCMFormatFloat32BE:
		return float64(math.Float32frombits(binary.BigEndian.Uint32(p)))
	case types.PCMFormatFloat64LE:
		return math.Float64frombits(binary.LittleEndian.Uint64(p))
	case types.PCMFormatFloat64BE:
		return math.Float64frombits(binary.BigEndian.Uint64(p))
	default:
		panic(fmt.Sprintf("unknown format: %v", f))
	}
}

func setFloat64(f types.PCMFormat, p []byte, v float64) {
	switch f {
	case types.PCMFormatU8:
		p[0] = byte(math.Round(v*128 + 128))
	case types.PCMFormatS16LE:
		binary.LittleEndian.PutUint16(p, uint16(int16(math.Round(v*32768))))
	case types.PCMFormatS16BE:
		binary.BigEndian.PutUint16(p, uint16(int16(math.Round(v*32768))))
	case types.PCMFormatS24LE:
		val := int32(math.Round(v * 8388608))
		if val > 8388607 {
			val = 8388607
		}
		if val < -8388608 {
			val = -8388608
		}
		p[0] = byte(val)
		p[1] = byte(val >> 8)
		p[2] = byte(val >> 16)
	case types.PCMFormatS24BE:
		val := int32(math.Round(v * 8388608))
		if val > 8388607 {
			val = 8388607
		}
		if val < -8388608 {
			val = -8388608
		}
		p[0] = byte(val >> 16)
		p[1] = byte(val >> 8)
		p[2] = byte(val)
	case types.PCMFormatS32LE:
		binary.LittleEndian.PutUint32(p, uint32(int32(math.Round(v*2147483648))))
	case types.PCMFormatS32BE:
		binary.BigEndian.PutUint32(p, uint32(int32(math.Round(v*2147483648))))
	case types.PCMFormatS64LE:
		binary.LittleEndian.PutUint64(p, uint64(int64(math.Round(v*9223372036854775808))))
	case types.PCMFormatS64BE:
		binary.BigEndian.PutUint64(p, uint64(int64(math.Round(v*9223372036854775808))))
	case types.PCMFormatFloat32LE:
		binary.LittleEndian.PutUint32(p, math.Float32bits(float32(v)))
	case types.PCMFormatFloat32BE:
		binary.BigEndian.PutUint32(p, math.Float32bits(float32(v)))
	case types.PCMFormatFloat64LE:
		binary.LittleEndian.PutUint64(p, math.Float64bits(v))
	case types.PCMFormatFloat64BE:
		binary.BigEndian.PutUint64(p, math.Float64bits(v))
	default:
		panic(fmt.Sprintf("unknown format: %v", f))
	}
}

var _ io.Reader = (*Resampler)(nil)

func NewResampler(
	inFormat Format,
	inReader io.Reader,
	outFormat Format,
) (*Resampler, error) {
	r := &Resampler{
		inReader:  inReader,
		inFormat:  inFormat,
		outFormat: outFormat,
	}
	err := r.init()
	if err != nil {
		return nil, fmt.Errorf("unable to initialize a resampler from %#+v to %#+v: %w", inFormat, outFormat, err)
	}
	return r, nil
}

func (r *Resampler) init() error {
	r.inSampleSize = uint(r.inFormat.PCMFormat.Size())
	r.outSampleSize = uint(r.outFormat.PCMFormat.Size())

	r.inNumAvg = 1
	r.outNumRepeat = 1
	if r.inFormat.Channels != r.outFormat.Channels {
		switch {
		case r.inFormat.Channels == 1:
			r.outNumRepeat = uint(r.outFormat.Channels)
		case r.outFormat.Channels == 1:
			r.inNumAvg = uint(r.inFormat.Channels)
		default:
			return fmt.Errorf("do not know how to convert %d channels to %d", r.inFormat.Channels, r.outFormat.Channels)
		}
	}

	sampleRateAdjust := float64(r.outFormat.SampleRate) / float64(r.inFormat.SampleRate)
	r.outDistanceStep = uint64(float64(distanceStep) / sampleRateAdjust)

	r.inDistance = 0
	r.outDistance = 0

	return nil
}

func (r *Resampler) Read(p []byte) (int, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	maxOutChunks := uint64(len(p)) / uint64(r.outSampleSize) / uint64(r.outNumRepeat)
	if maxOutChunks == 0 {
		return 0, nil
	}

	chunksToRead := uint64(float64(maxOutChunks) * float64(r.inFormat.SampleRate) / float64(r.outFormat.SampleRate))
	if chunksToRead == 0 {
		chunksToRead = 1
	}
	bytesToRead := uint64(chunksToRead) * uint64(r.inSampleSize) * uint64(r.inNumAvg)
	if cap(r.buffer) < int(bytesToRead) {
		r.buffer = make([]byte, bytesToRead)
	} else {
		r.buffer = r.buffer[:bytesToRead]
	}
	n, err := r.inReader.Read(r.buffer)
	r.buffer = r.buffer[:n]

	if n > 0 && n%int(r.inSampleSize*r.inNumAvg) != 0 {
		return 0, fmt.Errorf("read a number of bytes (%d) that is not a multiple of %d", n, r.inSampleSize*r.inNumAvg)
	}
	chunksRead := uint64(n) / uint64(r.inSampleSize) / uint64(r.inNumAvg)

	dstChunkIdx := uint64(0)
	srcChunkIdx := uint64(0)
	for srcChunkIdx < chunksRead && dstChunkIdx < maxOutChunks {
		// If we are too far ahead in input distance, skip input samples
		for r.inDistance < r.outDistance && srcChunkIdx < chunksRead {
			srcChunkIdx++
			r.inDistance += distanceStep
		}
		if srcChunkIdx >= chunksRead {
			break
		}

		// Read input sample
		idxSrc := srcChunkIdx * uint64(r.inSampleSize) * uint64(r.inNumAvg)
		var sum float64
		for channelIdx := uint64(0); channelIdx < uint64(r.inNumAvg); channelIdx++ {
			sum += getFloat64(r.inFormat.PCMFormat, r.buffer[idxSrc+channelIdx*uint64(r.inSampleSize):])
		}
		val := sum / float64(r.inNumAvg)

		// Write output sample (possibly repeated)
		for dstChunkIdx < maxOutChunks && r.outDistance <= r.inDistance {
			for repeatIdx := uint64(0); repeatIdx < uint64(r.outNumRepeat); repeatIdx++ {
				idxDst := (dstChunkIdx*uint64(r.outNumRepeat) + repeatIdx) * uint64(r.outSampleSize)
				setFloat64(r.outFormat.PCMFormat, p[idxDst:], val)
			}
			dstChunkIdx++
			r.outDistance += r.outDistanceStep
		}

		srcChunkIdx++
		r.inDistance += distanceStep
	}

	return int(dstChunkIdx * uint64(r.outSampleSize) * uint64(r.outNumRepeat)), err
}
