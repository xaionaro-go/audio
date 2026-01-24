// Package gccphat implements an audio synchronization algorithm using
// Generalized Cross-Correlation with Phase Transform (GCC-PHAT).
//
// The algorithm calculates the time delay between two signals by
// looking at their cross-correlation in the frequency domain. By
// normalizing the magnitude (the Phase Transform), it becomes
// robust against variations in volume and certain types of noise,
// focusing only on the phase information that indicates the delay.
package gccphat

import (
	"context"
	"fmt"

	"github.com/mjibson/go-dsp/fft"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/syncer"
)

type Syncer struct {
	EncodingValue audio.Encoding
	ChannelsValue audio.Channel
	MinFreq       float64
	MaxFreq       float64
}

var _ syncer.Syncer = (*Syncer)(nil)

// NewSyncer initializes a new one-shot GCC-PHAT syncer.
func NewSyncer(
	encoding audio.Encoding,
	channels audio.Channel,
) (*Syncer, error) {
	if encoding == nil {
		return nil, fmt.Errorf("encoding is mandatory")
	}
	if channels <= 0 {
		return nil, fmt.Errorf("channels must be greater than 0: got %d", channels)
	}

	if pcm, ok := encoding.(audio.EncodingPCM); !ok || pcm.SampleRate == 0 {
		return nil, fmt.Errorf("sample rate is mandatory and could not be determined from encoding %T", encoding)
	}

	return &Syncer{
		EncodingValue: encoding,
		ChannelsValue: channels,
		// Reasonable defaults: 100Hz to 12000Hz captures most informative audio
		// while filtering out low-frequency rumble and high-frequency digital noise.
		MinFreq: 100,
		MaxFreq: 12000,
	}, nil
}

func (s *Syncer) Close() error {
	return nil
}

func (s *Syncer) Encoding(
	ctx context.Context,
) (audio.Encoding, error) {
	return s.EncodingValue, nil
}

func (s *Syncer) Channels(
	ctx context.Context,
) (audio.Channel, error) {
	return s.ChannelsValue, nil
}

func (s *Syncer) CalculateShiftBetween(
	ctx context.Context,
	referenceTrack []byte,
	comparisonTracks ...[]byte,
) ([]syncer.ShiftResult, error) {
	refSamples, err := ToSamples(s.EncodingValue, s.ChannelsValue, referenceTrack)
	if err != nil {
		return nil, fmt.Errorf("failed to convert reference track to samples: %w", err)
	}

	results := make([]syncer.ShiftResult, len(comparisonTracks))
	for i, comparisonTrack := range comparisonTracks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		compSamples, err := ToSamples(s.EncodingValue, s.ChannelsValue, comparisonTrack)
		if err != nil {
			return nil, fmt.Errorf("failed to convert comparison track %d to samples: %w", i, err)
		}

		// Determine the FFT size: the next power of two of (n1 + n2 - 1)
		// to avoid circular convolution artifacts.
		n1 := len(refSamples)
		n2 := len(compSamples)
		n := 1
		for n < n1+n2-1 {
			n <<= 1
		}

		fref := make([]complex128, n)
		fcomp := make([]complex128, n)

		// Pad with zeros to the FFT size.
		for j := 0; j < n1; j++ {
			fref[j] = complex(refSamples[j], 0)
		}
		for j := 0; j < n2; j++ {
			fcomp[j] = complex(compSamples[j], 0)
		}

		// Transform signals to the frequency domain (Forward FFT).
		ffref := fft.FFT(fref)
		ffcomp := fft.FFT(fcomp)

		encPCM, ok := s.EncodingValue.(audio.EncodingPCM)
		if !ok || encPCM.SampleRate == 0 {
			return nil, fmt.Errorf("sample rate is required for band limiting")
		}
		sampleRate := float64(encPCM.SampleRate)

		shift, confidence, err := CrossCorrelate(ffref, ffcomp, sampleRate, s.MinFreq, s.MaxFreq)
		if err != nil {
			return nil, fmt.Errorf("failed to cross-correlate track %d: %w", i, err)
		}
		results[i] = syncer.ShiftResult{
			Shift:      shift,
			Confidence: confidence,
		}
	}
	return results, nil
}
