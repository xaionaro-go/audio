//go:build rnnoise
// +build rnnoise

package rnnoise

import (
	"context"
	"fmt"
	"math"
	"sync"
	"unsafe"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/audio/planar"
	"github.com/xaionaro-go/audio/pkg/noisesuppression"
	"github.com/xaionaro-go/observability"
)

/*
#cgo pkg-config: rnnoise
#cgo CFLAGS: -march=native
#include <rnnoise.h>
*/
import "C"

const (
	debugByPassProcessingFrames = false
)

type RNNoise struct {
	Locker        sync.Mutex
	DenoiseStates []*C.DenoiseState
	ChannelCount  audio.Channel
	Buffer        []byte
}

var _ noisesuppression.NoiseSuppression = (*RNNoise)(nil)

var frameSize int

func init() {
	frameSize = int(C.rnnoise_get_frame_size())
}

func New(
	channels audio.Channel,
) (*RNNoise, error) {
	var denoiseState []*C.DenoiseState
	for ch := 0; ch < int(channels); ch++ {
		denoiseState = append(denoiseState, C.rnnoise_create(nil))
	}
	return &RNNoise{
		DenoiseStates: denoiseState,
		ChannelCount:  channels,
	}, nil
}

func (s *RNNoise) Close() error {
	if s.DenoiseStates == nil {
		return fmt.Errorf("double-free attempt")
	}
	for _, denoiseState := range s.DenoiseStates {
		C.rnnoise_destroy(denoiseState)
	}
	s.DenoiseStates = nil
	return nil
}

func (s *RNNoise) Encoding(ctx context.Context) (audio.Encoding, error) {
	var pcmFormat audio.PCMFormat
	switch getEndian() {
	case endianBig:
		pcmFormat = audio.PCMFormatFloat32BE
	case endianLittle:
		pcmFormat = audio.PCMFormatFloat32LE
	default:
		return nil, fmt.Errorf("unable to detect endianness of this computer")
	}
	return audio.EncodingPCM{
		PCMFormat:  pcmFormat,
		SampleRate: 48_000,
	}, nil
}

func (s *RNNoise) Channels(ctx context.Context) (audio.Channel, error) {
	return s.ChannelCount, nil
}

var floatSize = unsafe.Sizeof(float32(0))

func chunkSize(channel audio.Channel) uint {
	return uint(channel) * uint(frameSize) * uint(floatSize)
}

func (s *RNNoise) ChunkSize() uint {
	return chunkSize(s.ChannelCount)
}

func (s *RNNoise) SuppressNoise(ctx context.Context, input []byte, outputVoice []byte) (_ret float64, _err error) {
	logger.Tracef(ctx, "SuppressNoise, len:%d", len(input))
	defer func() { logger.Tracef(ctx, "/SuppressNoise, len:%d: %v", len(input), _err) }()

	if len(input)%int(floatSize) != 0 {
		return 0, fmt.Errorf("the size of the input is not a multiple of size float32: %d %% %d != 0", len(input), floatSize)
	}
	if len(input) != len(outputVoice) {
		return 0, fmt.Errorf("lengths of input and output slices are not equal: %d != %d", len(input), len(outputVoice))
	}
	if len(input) < int(s.ChunkSize()) {
		return 0, fmt.Errorf("the size of the input is too small: %d < %d", len(input), s.ChunkSize())
	}
	if len(input)%int(s.ChunkSize()) != 0 {
		return 0, fmt.Errorf("the size of the input is not a multiple of ChunkSize: %d %% %d != 0", len(input), int(s.ChunkSize()))
	}

	s.Locker.Lock()
	defer s.Locker.Unlock()
	if len(s.Buffer) < len(input) {
		s.Buffer = make([]byte, len(input))
	}

	if s.ChannelCount == 1 {
		gain(s.Buffer[:len(input)], input)
		v := noiseSuppressOneChannel(ctx, s.DenoiseStates[0], s.Buffer[:len(input)], outputVoice)
		ungain(outputVoice)
		return v, nil
	}

	v := noiseSuppressMultipleChannels(ctx, s.DenoiseStates, input, outputVoice, s.Buffer[:len(input)])
	return v, nil
}

func noiseSuppressOneChannel(ctx context.Context, denoiseState *C.DenoiseState, input []byte, outputVoice []byte) float64 {
	var maxVADProb float64
	logger.Tracef(ctx, "noiseSuppressOneChannel, len:%d", len(input))
	chunkSize := chunkSize(1)
	for len(input) > 0 {
		if debugByPassProcessingFrames {
			copy(outputVoice[:chunkSize], input[:chunkSize])
		} else {
			vadProb := C.rnnoise_process_frame(
				denoiseState,
				(*C.float)(unsafe.Pointer(unsafe.SliceData(outputVoice[:chunkSize]))),
				(*C.float)(unsafe.Pointer(unsafe.SliceData(input[:chunkSize]))),
			)
			if float64(vadProb) > maxVADProb {
				maxVADProb = float64(vadProb)
			}
		}
		input = input[chunkSize:]
		outputVoice = outputVoice[chunkSize:]
	}
	return maxVADProb
}

func noiseSuppressMultipleChannels(
	ctx context.Context,
	denoiseStates []*C.DenoiseState,
	input []byte,
	outputVoice []byte,
	buffer []byte,
) float64 {
	if len(input) != len(outputVoice) {
		panic("len(input) != len(outputVoice)")
	}
	if len(input) != len(buffer) {
		panic("len(input) != len(buffer)")
	}

	channels := len(denoiseStates)
	err := planar.Planarize(audio.Channel(channels), uint(floatSize), buffer, input)
	if err != nil {
		panic(err)
	}
	gain(buffer, buffer)

	oneChanSize := len(buffer) / channels

	var locker sync.Mutex
	var maxVADProb float64
	var wg sync.WaitGroup
	for ch := 0; ch < channels; ch++ {
		denoiseState := denoiseStates[ch]
		data := buffer[ch*oneChanSize : (ch+1)*oneChanSize]
		wg.Add(1)
		observability.Go(ctx, func(ctx context.Context) {
			defer wg.Done()
			vadProb := noiseSuppressOneChannel(ctx, denoiseState, data, data)
			locker.Lock()
			defer locker.Unlock()
			if vadProb > maxVADProb {
				maxVADProb = vadProb
			}
		})
	}
	wg.Wait()

	ungain(outputVoice)
	err = planar.Unplanarize(audio.Channel(channels), uint(floatSize), outputVoice, buffer)
	if err != nil {
		panic(err)
	}

	return maxVADProb
}

func gain(dstBytes, srcBytes []byte) {
	src := unsafe.Slice((*float32)(unsafe.Pointer(&srcBytes[0])), len(srcBytes)/4)
	dst := unsafe.Slice((*float32)(unsafe.Pointer(&dstBytes[0])), len(dstBytes)/4)
	for idx := range src {
		dst[idx] = src[idx] * math.MaxInt16
	}
}

func ungain(buf []byte) {
	s := unsafe.Slice((*float32)(unsafe.Pointer(&buf[0])), len(buf)/4)
	for idx := range s {
		s[idx] /= math.MaxInt16
	}
}
