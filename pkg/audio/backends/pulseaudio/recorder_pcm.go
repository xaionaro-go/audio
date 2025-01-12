package pulseaudio

import (
	"fmt"
	"io"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

type RecorderPCM struct {
}

var _ types.RecorderPCM = (*RecorderPCM)(nil)

func NewRecorderPCM() RecorderPCM {
	return RecorderPCM{}
}

func (RecorderPCM) Ping() error {
	c, err := pulse.NewClient()
	if err != nil {
		return fmt.Errorf("unable to open a client to Pulse: %w", err)
	}
	defer c.Close()
	return nil
}

func (RecorderPCM) RecordPCM(
	sampleRate types.SampleRate,
	channels types.Channel,
	format types.PCMFormat,
	rawWriter io.Writer,
) (_ types.RecordStream, _err error) {
	writer, err := newPulseWriter(format, rawWriter)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize a writer for Pulse: %w", err)
	}

	c, err := pulse.NewClient()
	if err != nil {
		return nil, fmt.Errorf("unable to open a client to Pulse: %w", err)
	}
	defer func() {
		if _err != nil {
			c.Close()
		}
	}()

	chanMap := proto.ChannelMap{proto.ChannelMono}
	switch channels {
	case 1:
	case 2:
		chanMap = proto.ChannelMap{proto.ChannelLeft, proto.ChannelRight}
	default:
		return nil, fmt.Errorf("do not know how to configer %d channels", channels)
	}

	stream, err := c.NewRecord(
		writer,
		pulse.RecordSampleRate(int(sampleRate)),
		pulse.RecordChannels(chanMap),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize a playback: %w", err)
	}

	stream.Start()
	if stream.Error() != nil {
		return nil, fmt.Errorf("an error occurred during playback: %w", stream.Error())
	}

	return newRecordStream(c, stream), nil
}

type pulseWriter struct {
	pulseFormat byte
	io.Writer
}

func newPulseWriter(pcmFormat types.PCMFormat, writer io.Writer) (*pulseWriter, error) {
	var pulseFormat byte
	switch pcmFormat {
	case types.PCMFormatFloat32LE:
		pulseFormat = proto.FormatFloat32LE
	default:
		return nil, fmt.Errorf("received an unexpected format: %v", pcmFormat)
	}
	return &pulseWriter{
		pulseFormat: pulseFormat,
		Writer:      writer,
	}, nil
}

var _ pulse.Writer = (*pulseWriter)(nil)

func (r pulseWriter) Format() byte {
	return r.pulseFormat
}
