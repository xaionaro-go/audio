package fourier

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterpolateFourier_NoClicks(t *testing.T) {
	// Create a sine wave with a gap to be interpolated
	freq := 440.0
	sampleRate := 44100.0

	before := make([]float64, 2048)
	for i := range before {
		before[i] = math.Sin(2 * math.Pi * freq * float64(i) / sampleRate)
	}

	size := 441 // 10ms gap

	after := make([]float64, 2048)
	for i := range after {
		// Continuation after the gap
		after[i] = math.Sin(2 * math.Pi * freq * float64(i+len(before)+size) / sampleRate)
	}

	interpolator := New()
	interpolated := interpolator.Interpolate(before, after, size)
	require.Equal(t, size, len(interpolated))

	// Check continuity at the boundaries
	// We check if the jump at the boundary is similar to the normal sample-to-sample difference.

	// Typical difference between samples in the signal
	maxDiff := 0.0
	for i := 1; i < len(before); i++ {
		d := math.Abs(before[i] - before[i-1])
		if d > maxDiff {
			maxDiff = d
		}
	}

	// Boundary 1: before and interpolated
	d1 := math.Abs(interpolated[0] - before[len(before)-1])
	require.LessOrEqual(t, d1, maxDiff*1.5, "Value jump too large at before boundary")

	// Boundary 2: interpolated and after
	d2 := math.Abs(after[0] - interpolated[len(interpolated)-1])
	require.LessOrEqual(t, d2, maxDiff*1.5, "Value jump too large at after boundary")

	// Check internal differences in the interpolated part
	for i := 1; i < len(interpolated); i++ {
		d := math.Abs(interpolated[i] - interpolated[i-1])
		require.LessOrEqual(t, d, maxDiff*3.0, "Click detected within interpolated part at index %d", i)
	}
}

func BenchmarkInterpolate(b *testing.B) {
	sampleRate := 44100.0
	freq := 440.0
	before := make([]float64, 2048)
	for i := range before {
		before[i] = math.Sin(2 * math.Pi * freq * float64(i) / sampleRate)
	}

	durations := []struct {
		name string
		ms   int
	}{
		{"10ms", 10},
		{"100ms", 100},
	}

	interpolator := New()

	for _, d := range durations {
		gapLen := int(float64(d.ms) * sampleRate / 1000.0)
		after := make([]float64, 2048)
		for i := range after {
			after[i] = math.Sin(2 * math.Pi * freq * float64(i+len(before)+gapLen) / sampleRate)
		}

		b.Run(d.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = interpolator.Interpolate(before, after, gapLen)
			}
		})
	}
}
