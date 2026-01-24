package gccphat

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/cmplx"

	"github.com/mjibson/go-dsp/fft"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/audio/resampler"
)

func ToSamples(
	encoding audio.Encoding,
	channels audio.Channel,
	data []byte,
) ([]float64, error) {
	encPCM, ok := encoding.(audio.EncodingPCM)
	if !ok {
		return nil, fmt.Errorf("unsupported encoding type: %T", encoding)
	}

	sampleRate := encPCM.SampleRate
	if sampleRate == 0 {
		return nil, fmt.Errorf("sample rate is mandatory")
	}

	inFmt := resampler.Format{
		Channels:   channels,
		SampleRate: sampleRate,
		PCMFormat:  encPCM.PCMFormat,
	}
	outFmt := resampler.Format{
		Channels:   1, // Mono for cross-correlation
		SampleRate: sampleRate,
		PCMFormat:  audio.PCMFormatFloat64LE,
	}

	r, err := resampler.NewResampler(inFmt, bytes.NewReader(data), outFmt)
	if err != nil {
		return nil, err
	}

	numSamples := len(data) / (int(encPCM.BytesPerSample()) * int(channels))
	out := make([]byte, numSamples*8)
	n, err := r.Read(out)
	if err != nil && err != io.EOF {
		return nil, err
	}

	samples := make([]float64, n/8)
	for i := range samples {
		samples[i] = math.Float64frombits(binary.LittleEndian.Uint64(out[i*8:]))
	}
	return samples, nil
}

// CrossCorrelate calculates the sample shift of 'fcomp' relative to 'fref' using GCC-PHAT.
// The fref and fcomp slices are expected to be the FFTs of the reference and comparison snippets.
// Both must have the same length N.
//
// Arguments:
// - sampleRate: Used to calculate frequency bin indices for band limiting.
// - minFreq: Minimum frequency to consider (Hz). Use 0 for no limit.
// - maxFreq: Maximum frequency to consider (Hz). Use 0 or >sampleRate/2 for no limit.
//
// Returns (shift, confidence, error). A positive shift means 'comp' leads 'ref'.
func CrossCorrelate(fref, fcomp []complex128, sampleRate float64, minFreq, maxFreq float64) (float64, float64, error) {
	if sampleRate <= 0 {
		return 0, 0, fmt.Errorf("sampleRate must be positive: got %v", sampleRate)
	}
	if len(fref) != len(fcomp) {
		return 0, 0, fmt.Errorf("fref and fcomp must have same length: %d != %d", len(fref), len(fcomp))
	}
	n := len(fref)

	// Frequency range to bin conversion
	binMin := 0
	binMax := n / 2
	if minFreq > 0 {
		binMin = int(minFreq * float64(n) / sampleRate)
	}
	if maxFreq > 0 && maxFreq < sampleRate/2 {
		binMax = int(maxFreq * float64(n) / sampleRate)
	}

	// Compute the cross-power spectrum with Phase Transform (PHAT).
	res := make([]complex128, n)

	// To make PHAT more robust, we only whiten bins that have energy
	// above a certain threshold relative to the maximum energy.
	maxMag := 0.0
	for i := 0; i < n; i++ {
		mag := cmplx.Abs(fcomp[i] * cmplx.Conj(fref[i]))
		if mag > maxMag {
			maxMag = mag
		}
	}
	threshold := maxMag * 0.001 // 60dB down

	activeBins := 0
	for i := 0; i < n; i++ {
		idx := i
		if i > n/2 {
			idx = n - i
		}

		if idx < binMin || idx > binMax {
			res[i] = 0
			continue
		}

		prod := fcomp[i] * cmplx.Conj(fref[i])
		mag := cmplx.Abs(prod)
		if mag > threshold && mag > 1e-12 {
			res[i] = prod / complex(mag, 0)
			activeBins++
		} else {
			res[i] = 0
		}
	}

	if activeBins == 0 {
		return 0, 0, nil
	}

	// Transform back to the time domain (Inverse FFT).
	timeDomain := fft.IFFT(res)

	// Find the peak in the cross-correlation result.
	maxVal := -1.0
	maxIdx := 0
	for i := range n {
		// The result of GCC-PHAT should be real-ish, but anyway we take Abs.
		val := cmplx.Abs(timeDomain[i])
		if val > maxVal {
			maxVal = val
			maxIdx = i
		}
	}

	// peak shift where comp(t) = ref(t-shift).
	shift := float64(maxIdx)
	if shift > float64(n/2) {
		shift -= float64(n)
	}

	// Sub-sample interpolation (Parabolic)
	if maxIdx > 0 && maxIdx < n-1 {
		y1 := cmplx.Abs(timeDomain[maxIdx-1])
		y2 := maxVal
		y3 := cmplx.Abs(timeDomain[maxIdx+1])

		denom := (y1 - 2*y2 + y3)
		if math.Abs(denom) > 1e-12 {
			delta := (y1 - y3) / (2 * denom)
			shift += delta
		}
	}

	// Resulting confidence is the peak magnitude normalized by the fraction of energy.
	// In a perfect match, maxVal in the time domain should be activeBins / n because
	// we have activeBins samples of magnitude 1.0 in the frequency domain, and IFFT
	// divides by N.
	// So confidence = maxVal * N / activeBins.
	confidence := 0.0
	if activeBins > 0 {
		confidence = maxVal * float64(n) / float64(activeBins)
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	// If comp(t) = ref(t-shift), shift > 0 means comp is LATER than ref (comp lags ref).
	// We want comp leads ref -> positive.
	// So we return -shift.
	return -shift, confidence, nil
}
