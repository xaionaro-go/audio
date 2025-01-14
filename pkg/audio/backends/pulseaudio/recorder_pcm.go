package pulseaudio

import (
	"context"
	"fmt"
	"io"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

type RecorderPCM struct {
	PulseClient *pulse.Client
}

var _ types.RecorderPCM = (*RecorderPCM)(nil)

func NewRecorderPCM() (*RecorderPCM, error) {
	c, err := pulse.NewClient()
	if err != nil {
		return nil, fmt.Errorf("unable to open a client to Pulse: %w", err)
	}
	return &RecorderPCM{
		PulseClient: c,
	}, nil
}

func (r *RecorderPCM) Close() error {
	r.PulseClient.Close()
	return nil
}

func (r *RecorderPCM) Ping(context.Context) error {
	_, err := r.PulseClient.DefaultSource()
	return err
}

func (r *RecorderPCM) RecordPCM(
	ctx context.Context,
	sampleRate types.SampleRate,
	channels types.Channel,
	format types.PCMFormat,
	rawWriter io.Writer,
) (_ types.RecordStream, _err error) {
	writer, err := newPulseWriter(format, rawWriter)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize a writer for Pulse: %w", err)
	}

	chanMap := proto.ChannelMap{proto.ChannelMono}
	switch channels {
	case 1:
	case 2:
		chanMap = proto.ChannelMap{proto.ChannelLeft, proto.ChannelRight}
	default:
		return nil, fmt.Errorf("do not know how to configer %d channels", channels)
	}

	stream, err := r.PulseClient.NewRecord(
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

	return newRecordStream(r.PulseClient, stream), nil
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
