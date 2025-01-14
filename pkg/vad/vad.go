package vad

import (
	"context"
	"io"
	"time"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type VAD interface {
	io.Closer

	Encoding(context.Context) (audio.Encoding, error)
	Channels(context.Context) (audio.Channel, error)

	FindNextVoice(
		_ context.Context,
		samples []byte,
		confidenceThreshold float64,
		minDuration time.Duration,
	) (float64, time.Duration, error)
}
