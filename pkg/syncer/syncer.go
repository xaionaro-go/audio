package syncer

import (
	"context"
	"time"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type Syncer interface {
	audio.AbstractAnalyzer

	CalculateShiftBetween(
		ctx context.Context,
		referenceTrack []byte,
		comparisonTracks ...[]byte,
	) ([]time.Duration, error)
}

/* for easier copy&paste:

func () Close() error {
}

func () Encoding(
	ctx context.Context,
) (audio.Encoding, error) {
}

func () Channels(
	ctx context.Context,
) (audio.Channel, error) {
}

func () CalculateShiftBetween(
	ctx context.Context,
	referenceTrack []byte,
	comparisonTracks ...[]byte,
) ([]time.Duration, error) {
}

*/
