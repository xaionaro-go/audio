package portaudio

import (
	"context"
	"fmt"
	"io"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/gordonklaus/portaudio"
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

type RecorderPCM struct {
}

var _ types.RecorderPCM = (*RecorderPCM)(nil)

func NewRecorderPCM() (*RecorderPCM, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}
	return &RecorderPCM{}, nil
}

func (*RecorderPCM) Close() error {
	return nil
}

func (*RecorderPCM) Ping(
	ctx context.Context,
) error {
	info, err := portaudio.DefaultInputDevice()
	if err != nil {
		return err
	}
	logger.Debugf(ctx, "device info: %#+v", info)

	if devices, err := portaudio.Devices(); err == nil {
		for idx, device := range devices {
			logger.Tracef(ctx, "devices[%d]: %#+v", idx, device)
		}
	}
	return nil
}

func (*RecorderPCM) RecordPCM(
	ctx context.Context,
	sampleRate types.SampleRate,
	channels types.Channel,
	format types.PCMFormat,
	writer io.Writer,
) (types.RecordStream, error) {
	var (
		s   *RecordPCMStream
		err error
	)
	switch format {
	case types.PCMFormatU8:
		s, err = newRecordPCMStream[uint8](ctx, sampleRate, channels)
	case types.PCMFormatS16LE:
		s, err = newRecordPCMStream[int16](ctx, sampleRate, channels)
	case types.PCMFormatFloat32LE:
		s, err = newRecordPCMStream[float32](ctx, sampleRate, channels)
	case types.PCMFormatS32LE:
		s, err = newRecordPCMStream[int32](ctx, sampleRate, channels)
	case types.PCMFormatFloat64LE:
		s, err = newRecordPCMStream[float64](ctx, sampleRate, channels)
	case types.PCMFormatS64LE:
		s, err = newRecordPCMStream[int64](ctx, sampleRate, channels)
	default:
		return nil, fmt.Errorf("do not know how to start a stream for PCM format %s", format)
	}

	if err := s.init(ctx, writer); err != nil {
		s.Close()
		return nil, fmt.Errorf("unable to post-initialize the stream: %w", err)
	}
	return s, err
}
