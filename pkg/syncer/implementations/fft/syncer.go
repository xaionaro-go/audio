package fft

import (
	"context"
	"time"

	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/syncer"
)

type Syncer struct{}

var _ syncer.Syncer = (*Syncer)(nil)

func NewSyncer() *Syncer {
	return &Syncer{}
}

func (s *Syncer) Close() error {
	return nil
}

func (s *Syncer) Encoding(
	ctx context.Context,
) (audio.Encoding, error) {
	panic("not implemented, yet")
}

func (s *Syncer) Channels(
	ctx context.Context,
) (audio.Channel, error) {
	panic("not implemented, yet")
}

func (s *Syncer) CalculateShiftBetween(
	ctx context.Context,
	referenceTrack []byte,
	comparisonTracks ...[]byte,
) ([]time.Duration, error) {
	panic("not implemented, yet")
}
