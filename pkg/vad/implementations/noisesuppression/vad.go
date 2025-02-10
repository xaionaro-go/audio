package noisesuppression

import (
	"context"
	"fmt"
	"time"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/noisesuppression"
	"github.com/xaionaro-go/audio/pkg/vad"
)

type VAD struct {
	noisesuppression.NoiseSuppression
	ChunkSize     uint64
	ChunkDuration time.Duration
	Buffer        []byte
}

var _ vad.VAD = (*VAD)(nil)

func NewVAD(
	ctx context.Context,
	noiseSuppression noisesuppression.NoiseSuppression,
	preferredGranularity time.Duration,
) (*VAD, error) {
	chunkSize := noiseSuppression.ChunkSize()
	channels, err := noiseSuppression.Channels(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get the amount of channels: %w", err)
	}
	encoding, err := noiseSuppression.Encoding(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get the encoding: %w", err)
	}
	encodingPCM, ok := encoding.(audio.EncodingPCM)
	if !ok {
		return nil, fmt.Errorf("noise suppression encoding is not PCM: %w", err)
	}
	preferredChunkSize := encoding.BytesForDuration(preferredGranularity) * uint64(channels)
	subChunks := (preferredChunkSize + uint64(chunkSize)/2) / uint64(chunkSize)
	if subChunks < 1 {
		subChunks = 1
	}
	chosenChunkSize := subChunks * uint64(chunkSize)
	chosenChunkSamples := chosenChunkSize / uint64(encoding.BytesPerSample()) / uint64(channels)
	chosenChunkDurationNS := uint64(1_000_000_000) * chosenChunkSamples / uint64(encodingPCM.SampleRate)
	chosenChunkDuration := time.Nanosecond * time.Duration(chosenChunkDurationNS)
	logger.Debugf(ctx, "resulting chunkSize:%d and chunkDuration:%v", chosenChunkSize, chosenChunkDuration)

	return &VAD{
		NoiseSuppression: noiseSuppression,
		ChunkSize:        chosenChunkSize,
		ChunkDuration:    chosenChunkDuration,
		Buffer:           make([]byte, chosenChunkSize),
	}, nil
}

func (v *VAD) FindNextVoice(
	ctx context.Context,
	samples []byte,
	confidenceThreshold float64,
	minDuration time.Duration,
) (float64, time.Duration, error) {
	if len(samples) == 0 {
		return 0, -1, nil
	}

	var maxConfidence float64

	var foundVoiceFor time.Duration
	firstVoiceDetection := time.Duration(-1)

	chunkSize := v.ChunkSize
	chunkDuration := v.ChunkDuration
	for pos := 0; ; pos++ {
		if len(samples) < int(chunkSize) {
			return maxConfidence, firstVoiceDetection, nil
		}
		frame := samples[:chunkSize]
		samples = samples[len(frame):]
		voiceConfidence, err := v.NoiseSuppression.SuppressNoise(ctx, frame, v.Buffer)
		if err != nil {
			return maxConfidence, firstVoiceDetection, err
		}

		if voiceConfidence > maxConfidence {
			maxConfidence = voiceConfidence
		}

		if voiceConfidence >= confidenceThreshold {
			foundVoiceFor += chunkDuration
			if firstVoiceDetection < 0 {
				firstVoiceDetection = chunkDuration * time.Duration(pos)
			}
		}

		if foundVoiceFor >= minDuration {
			return maxConfidence, firstVoiceDetection, nil
		}
	}
}
