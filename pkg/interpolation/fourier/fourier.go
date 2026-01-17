package fourier

import (
	"math"

	"github.com/brettbuddin/fourier"
	"github.com/xaionaro-go/audio/pkg/interpolation"
)

const (
	// MaxWindowSize is the maximum number of samples used for FFT analysis.
	// 1024 provides a good balance between frequency resolution and performance.
	MaxWindowSize = 1024

	// MinRequiredSamples is the minimum number of samples needed on each side
	// of the gap to perform a meaningful spectral analysis.
	MinRequiredSamples = 4

	// SieveSensitivity factor determines how far a spectral peak must stand
	// above the average noise floor to be considered significant.
	// A value of 2.5 is chosen to filter out room noise and low-level artifacts.
	SieveSensitivity = 2.5

	// SpectrumNormalization scales the magnitudes from a two-sided forward FFT
	// to their real-world amplitudes for synthesis.
	SpectrumNormalization = 2.0
)

type Interpolator struct{}

func New() interpolation.Interpolator {
	return &Interpolator{}
}

// Interpolate fills an audio gap using a bidirectional Spectral Sieve method.
//
// The algorithm works as follows:
//
// 1. Windowing: It extracts a window of samples immediately before and after the gap.
// The window size is optimized for Radix-2 FFT (power of two, capped by MaxWindowSize).
//
// 2. Spectral Sieve: For each window, it performs a Forward FFT. It then filters
// the frequency components using a "sieve" that identifies significant spectral peaks
// above a dynamic noise threshold. This isolates the tonal components of the signal
// from stochastic noise, preventing noisy artifacts in the extension.
//
// 3. Synthesis and Projection: It projects these tonal components into the gap area
// by synthesizing sine waves that maintain the phase and frequency of the identified
// peaks. This is done for both 'forward' (from the past) and 'backward' (from the future).
//
// 4. Cubic Cross-fade: The two projections are blended using a cubic weighting function
// (3t^2 - 2t^3), which ensures a smooth transition between the two spectral estimations
// and provides C1 continuity (matching first derivatives) at the center of the gap.
//
// 5. Boundary Trend Correction: Finally, it calculates the offset between the synthesized
// values and the actual samples at the exact boundary. A linear trend adjustment is applied
// to the entire interpolated segment to ensure there is zero discontinuity at the stitch points,
// effectively eliminating audible clicks.
func (i *Interpolator) Interpolate(before, after []float64, gapLen int) []float64 {
	if len(before) < MinRequiredSamples || len(after) < MinRequiredSamples {
		return make([]float64, gapLen)
	}

	result := make([]float64, gapLen)
	n := min(len(before), MaxWindowSize, len(after))
	n = largestPowerOfTwo(n)

	windowBefore := before[len(before)-n:]
	windowAfter := after[:n]

	forward := extendSpectralSieve(windowBefore, gapLen, true)
	backward := extendSpectralSieve(windowAfter, gapLen, false)

	vStart := windowBefore[len(windowBefore)-1]
	vEnd := windowAfter[0]

	for i := range gapLen {
		t := float64(i+1) / float64(gapLen+1)
		w := t * t * (3 - 2*t) // cubic

		val := (1-w)*forward[i] + w*backward[i]

		startDiff := forward[0] - vStart
		endDiff := backward[gapLen-1] - vEnd
		val -= (1-w)*startDiff + w*endDiff

		result[i] = val
	}

	return result
}

func largestPowerOfTwo(n int) int {
	p := 1
	for p*2 <= n {
		p *= 2
	}
	return p
}

func extendSpectralSieve(samples []float64, gapLen int, forward bool) []float64 {
	n := len(samples)
	coeffs := make([]complex128, n)
	for i, v := range samples {
		coeffs[i] = complex(v, 0)
	}
	if err := fourier.Forward(coeffs); err != nil {
		// Fallback or handle error
		return make([]float64, gapLen)
	}

	magnitudes := make([]float64, len(coeffs))
	for i, c := range coeffs {
		magnitudes[i] = math.Hypot(real(c), imag(c))
	}

	var threshold float64
	for _, m := range magnitudes {
		threshold += m
	}
	threshold = (threshold / float64(len(magnitudes))) * SieveSensitivity

	type peak struct {
		idx   int
		coeff complex128
	}
	var peaks []peak
	for i := 1; i < len(coeffs)/2; i++ {
		if magnitudes[i] > threshold && magnitudes[i] > magnitudes[i-1] && magnitudes[i] > magnitudes[i+1] {
			peaks = append(peaks, peak{i, coeffs[i]})
		}
	}

	result := make([]float64, gapLen)
	invN := 1.0 / float64(n)
	for i := range gapLen {
		var t float64
		if forward {
			t = float64(n + i)
		} else {
			t = float64(i - gapLen)
		}

		var sum float64
		for _, p := range peaks {
			phase := 2.0 * math.Pi * float64(p.idx) * t * invN
			mag := magnitudes[p.idx] * SpectrumNormalization * invN
			origPhase := math.Atan2(imag(p.coeff), real(p.coeff))
			sum += mag * math.Cos(phase+origPhase)
		}
		// DC component
		sum += real(coeffs[0]) * invN
		result[i] = sum
	}
	return result
}
