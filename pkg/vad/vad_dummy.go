package vad

import (
	"context"
	"time"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type Dummy struct {
	EncodingValue audio.Encoding
	ChannelsValue audio.Channel
}

var _ VAD = (*Dummy)(nil)

func NewDummy(
	encoding audio.Encoding,
	channels audio.Channel,
) *Dummy {
	return &Dummy{
		EncodingValue: encoding,
		ChannelsValue: channels,
	}
}

func (vad *Dummy) Close() error {
	return nil
}

func (vad *Dummy) Encoding(context.Context) (audio.Encoding, error) {
	return vad.EncodingValue, nil
}

func (vad *Dummy) Channels(context.Context) (audio.Channel, error) {
	return vad.ChannelsValue, nil
}

func (vad *Dummy) FindNextVoice(
	_ context.Context,
	samples []byte,
	confidenceThreshold float64,
	minDuration time.Duration,
) (float64, time.Duration, error) {
	return 1, 0, nil
}
