package fft

import (
	"context"

	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/syncer"

	"github.com/brettbuddin/fourier"
)

type Syncer struct {
	EncodingValue audio.Encoding
	ChannelsValue audio.Channel
}

var _ syncer.Syncer = (*Syncer)(nil)

func NewSyncer(
	encoding audio.Encoding,
	channels audio.Channel,
) *Syncer {
	return &Syncer{
		EncodingValue: encoding,
		ChannelsValue: channels,
	}
}

func (s *Syncer) Close() error {
	return nil
}

func (s *Syncer) Encoding(
	ctx context.Context,
) (audio.Encoding, error) {
	return s.EncodingValue, nil
}

func (s *Syncer) Channels(
	ctx context.Context,
) (audio.Channel, error) {
	return s.ChannelsValue, nil
}

func (s *Syncer) CalculateShiftBetween(
	ctx context.Context,
	referenceTrack []byte,
	comparisonTracks ...[]byte,
) ([]int, error) {
	fourier.Forward(nil)
	panic("not implemented, yet")
}
