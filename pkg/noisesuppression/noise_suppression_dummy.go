package noisesuppression

import (
	"context"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type Dummy struct {
	EncodingValue audio.Encoding
	ChannelsValue audio.Channel
}

var _ NoiseSuppression = (*Dummy)(nil)

func NewDummy(
	encoding audio.Encoding,
	channels audio.Channel,
) *Dummy {
	return &Dummy{
		EncodingValue: encoding,
		ChannelsValue: channels,
	}
}

func (s *Dummy) Close() error {
	return nil
}

func (s *Dummy) Encoding(context.Context) (audio.Encoding, error) {
	return s.EncodingValue, nil
}

func (s *Dummy) Channels(context.Context) (audio.Channel, error) {
	return s.ChannelsValue, nil
}

func (*Dummy) ChunkSize() uint {
	return 0
}

func (*Dummy) SuppressNoise(context.Context, []byte, []byte) (float64, error) {
	return 1, nil
}
