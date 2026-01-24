package syncer

import (
	"context"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type ShiftResult struct {
	SampleOffset int64   // Position in the comparison stream (for streaming syncer)
	Shift        float64 // Delay relative to reference (positive means comparison is ahead)
	Confidence   float64 // Confidence score (0..1)
}

type Syncer interface {
	audio.AbstractAnalyzer

	// CalculateShiftBetween returns the amount of samples que
	// needs to be shifted by, to get a comparison track synced
	// with the reference track. It also returns a confidence
	// score (0..1) for each result.
	CalculateShiftBetween(
		ctx context.Context,
		referenceTrack []byte,
		comparisonTracks ...[]byte,
	) ([]ShiftResult, error)
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
// with the reference track. It also returns a confidence
// score (0..1) for each result.
func () CalculateShiftBetween(
	ctx context.Context,
	referenceTrack []byte,
	comparisonTracks ...[]byte,
) ([]int, []float64, error) {
}

*/
