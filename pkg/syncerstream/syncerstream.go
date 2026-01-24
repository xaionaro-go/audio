package syncerstream

import (
	"context"

	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/syncer"
)

type SyncerStream interface {
	audio.AbstractAnalyzer

	// PushReference feeds audio data for the reference signal.
	PushReference(ctx context.Context, data []byte) error

	// PushComparison feeds audio data for the signal to be synced.
	// Returns detected shifts for the specific track.
	PushComparison(ctx context.Context, trackID int, data []byte) ([]syncer.ShiftResult, error)
}

type Factory interface {
	NewSyncer(encoding audio.Encoding, channels audio.Channel) (SyncerStream, error)
}
