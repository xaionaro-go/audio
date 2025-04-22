package noisesuppression

import (
	"context"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type NoiseSuppression interface {
	audio.AbstractAnalyzer

	ChunkSize() uint
	SuppressNoise(ctx context.Context, input []byte, outputVoice []byte) (float64, error)
}
