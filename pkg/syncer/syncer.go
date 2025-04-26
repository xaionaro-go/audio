package syncer

import (
	"context"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type Syncer interface {
	audio.AbstractAnalyzer

	// CalculateShiftBetween returns the amount of samples that
	// needs to be shifted by, to get a comparison track synced
	// with the reference track.
	CalculateShiftBetween(
		ctx context.Context,
		referenceTrack []byte,
		comparisonTracks ...[]byte,
	) ([]int, error)
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

// CalculateShiftBetween returns the amount of samples that
// needs to be shifted by, to get a comparison track synced
// with the reference track.
func () CalculateShiftBetween(
	ctx context.Context,
	referenceTrack []byte,
	comparisonTracks ...[]byte,
) ([]int, error) {
}

*/
