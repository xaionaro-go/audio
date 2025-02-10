package noisesuppression

import (
	"context"
	"io"

	"github.com/xaionaro-go/audio/pkg/audio"
)

type NoiseSuppression interface {
	io.Closer

	Encoding(context.Context) (audio.Encoding, error)
	Channels(context.Context) (audio.Channel, error)
	ChunkSize() uint

	SuppressNoise(ctx context.Context, input []byte, outputVoice []byte) (float64, error)
}
