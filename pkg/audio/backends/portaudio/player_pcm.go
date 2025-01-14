package portaudio

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/gordonklaus/portaudio"
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

type PlayerPCM struct {
}

var _ types.PlayerPCM = (*PlayerPCM)(nil)

func NewPlayerPCM() (*PlayerPCM, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}
	return &PlayerPCM{}, nil
}

func (*PlayerPCM) Close() error {
	return nil
}

func (*PlayerPCM) Ping(
	ctx context.Context,
) error {
	info, err := portaudio.DefaultOutputDevice()
	if err != nil {
		return err
	}
	logger.Debugf(ctx, "device info: %#+v", info)
	return nil
}

func (*PlayerPCM) PlayPCM(
	ctx context.Context,
	sampleRate types.SampleRate,
	channels types.Channel,
	format types.PCMFormat,
	bufferSize time.Duration,
	rawReader io.Reader,
) (_ types.PlayStream, _err error) {
	var (
		s   *PlayPCMStream
		err error
	)
	switch format {
	case types.PCMFormatU8:
		s, err = newPlayPCMStream[uint8](ctx, sampleRate, channels, bufferSize)
	case types.PCMFormatS16LE:
		s, err = newPlayPCMStream[int16](ctx, sampleRate, channels, bufferSize)
	case types.PCMFormatFloat32LE:
		s, err = newPlayPCMStream[float32](ctx, sampleRate, channels, bufferSize)
	case types.PCMFormatS32LE:
		s, err = newPlayPCMStream[int32](ctx, sampleRate, channels, bufferSize)
	case types.PCMFormatFloat64LE:
		s, err = newPlayPCMStream[float64](ctx, sampleRate, channels, bufferSize)
	case types.PCMFormatS64LE:
		s, err = newPlayPCMStream[int64](ctx, sampleRate, channels, bufferSize)
	default:
		return nil, fmt.Errorf("do not know how to start a stream for PCM format %s", format)
	}

	if err := s.init(ctx, rawReader); err != nil {
		s.Close()
		return nil, fmt.Errorf("unable to post-initialize the stream: %w", err)
	}
	return s, err
}
