package vad

import (
	"context"
	"time"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type VAD interface {
	audio.AbstractAnalyzer

	FindNextVoice(
		_ context.Context,
		samples []byte,
		confidenceThreshold float64,
		minDuration time.Duration,
	) (float64, time.Duration, error)
}
