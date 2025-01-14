package libfvad

import (
	"context"
	"fmt"
	"time"

	"github.com/josharian/fvad"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/vad"
)

type VAD struct {
	*fvad.Detector
	SampleRate audio.SampleRate
}

var _ vad.VAD = (*VAD)(nil)

func NewVAD(
	sampleRate audio.SampleRate,
	sensitivityMode int,
) (*VAD, error) {
	detector := fvad.NewDetector()
	if err := detector.SetSampleRate(int(sampleRate)); err != nil {
		return nil, fmt.Errorf("unable to set the sample rate: %w", err)
	}
	if err := detector.SetMode(sensitivityMode); err != nil {
		return nil, fmt.Errorf("unable to set the sensitivity mode: %w", err)
	}
	return &VAD{
		SampleRate: sampleRate,
		Detector:   detector,
	}, nil
}

func (v *VAD) Close() error {
	v.Detector.Close()
	return nil
}

func (v *VAD) Encoding(context.Context) (audio.Encoding, error) {
	return v.EncodingNoErr(), nil
}

func (v *VAD) EncodingNoErr() audio.EncodingPCM {
	return audio.EncodingPCM{
		PCMFormat:  audio.PCMFormatS16LE,
		SampleRate: v.SampleRate,
	}
}

func (v *VAD) Channels(context.Context) (audio.Channel, error) {
	return v.ChannelsNoErr(), nil
}

func (*VAD) ChannelsNoErr() audio.Channel {
	return 1
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

	var foundVoiceFor time.Duration
	firstVoiceDetection := time.Duration(-1)

	// see the description of (*fvad.Detector).Process
	minPortion := v.pieceSize10Ms()
	midPortion := minPortion * 2
	maxPortion := minPortion * 3
	for pos := 0; ; pos++ {
		var frame []byte

		var curDuration time.Duration
		switch {
		case len(samples) >= int(maxPortion):
			frame = samples[:maxPortion]
			curDuration = 30 * time.Millisecond
		case len(samples) >= int(midPortion):
			frame = samples[:midPortion]
			curDuration = 20 * time.Millisecond
		case len(samples) >= int(minPortion):
			frame = samples[:minPortion]
			curDuration = 10 * time.Millisecond
		default:
			return 0, firstVoiceDetection, nil
		}
		samples = samples[len(frame):]
		procResult, err := v.Detector.Process(convertBytesToInt16Slice(frame))
		if err != nil {
			return 0, firstVoiceDetection, err
		}

		if procResult {
			foundVoiceFor += curDuration
			if firstVoiceDetection < 0 {
				firstVoiceDetection = 30 * time.Millisecond * time.Duration(pos)
			}
		}

		if foundVoiceFor >= minDuration {
			return 1, firstVoiceDetection, nil
		}
	}
}

func (v *VAD) pieceSize10Ms() uint32 {
	return uint32(2 * 80 * uint64(v.SampleRate) / 8000)
}
