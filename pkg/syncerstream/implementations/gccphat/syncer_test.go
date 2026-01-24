package gccphat

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xaionaro-go/audio/pkg/audio"
)

func float64ToBytes(data []float64) []byte {
	res := make([]byte, len(data)*8)
	for i, v := range data {
		binary.LittleEndian.PutUint64(res[i*8:], math.Float64bits(v))
	}
	return res
}

func TestSyncerStream_Push(t *testing.T) {
	encoding := audio.EncodingPCM{
		PCMFormat:  audio.PCMFormatFloat64LE,
		SampleRate: 44100,
	}
	channels := audio.Channel(1)
	windowSize := 1024
	hopSize := 512
	maxLag := 1024
	s, _ := NewSyncer(encoding, channels, windowSize, hopSize, maxLag, 0, 0)
	ctx := context.Background()

	// Create a reference signal (white noise)
	n := 8192
	ref := make([]float64, n)
	r := rand.New(rand.NewSource(42))
	for i := range ref {
		ref[i] = r.Float64()*2 - 1
	}

	// Create a comparison signal delayed by 10.0 samples
	shiftAmount := 10.0
	comp := make([]float64, n)
	for i := int(shiftAmount); i < n; i++ {
		comp[i] = ref[i-int(shiftAmount)]
	}

	// Feed reference
	err := s.PushReference(ctx, float64ToBytes(ref))
	assert.NoError(t, err)

	// Feed comparison
	results, err := s.PushComparison(ctx, 0, float64ToBytes(comp))
	assert.NoError(t, err)

	assert.Greater(t, len(results), 0)

	found := false
	for _, res := range results {
		fmt.Printf("Push result: shift=%v, conf=%v\n", res.Shift, res.Confidence)
		// Comparison is delayed, so shift should be -10.0
		if res.Confidence > 0.15 {
			assert.InDelta(t, -shiftAmount, res.Shift, 0.5)
			found = true
		}
	}
	assert.True(t, found)
}

func TestSyncerStream_LargeDelay(t *testing.T) {
	encoding := audio.EncodingPCM{
		PCMFormat:  audio.PCMFormatFloat64LE,
		SampleRate: 44100,
	}
	channels := audio.Channel(1)
	// Normal window size
	windowSize := 16384
	hopSize := 8192
	maxLag := 150000
	s, _ := NewSyncer(encoding, channels, windowSize, hopSize, maxLag, 0, 0)
	ctx := context.Background()

	// 3 seconds delay at 44.1kHz is 132300 samples
	shiftAmount := 3.0 * 44100.0
	n := int(shiftAmount) + 16384 + 8192

	ref := make([]float64, n)
	r := rand.New(rand.NewSource(42))
	for i := range ref {
		ref[i] = r.Float64()*2 - 1
	}

	comp := make([]float64, n)
	for i := int(shiftAmount); i < n; i++ {
		comp[i] = ref[i-int(shiftAmount)]
	}

	// Feed reference
	err := s.PushReference(ctx, float64ToBytes(ref))
	assert.NoError(t, err)

	// Feed comparison
	results, err := s.PushComparison(ctx, 0, float64ToBytes(comp))
	assert.NoError(t, err)

	found := false
	for _, res := range results {
		if res.Confidence > 0.15 {
			assert.InDelta(t, -shiftAmount, res.Shift, 0.5)
			found = true
		}
	}
	assert.True(t, found, "Should have found at least one high-confidence shift")
}

func BenchmarkSyncerStream(b *testing.B) {
	encoding := audio.EncodingPCM{
		PCMFormat:  audio.PCMFormatFloat64LE,
		SampleRate: 44100,
	}
	channels := audio.Channel(1)
	ctx := context.Background()

	// Parameters to vary
	windowSizes := []int{4096, 8192, 16384}
	numTracks := []int{1, 2, 4}
	maxLags := []int{0, 44100 * 3} // 0 means maxLag = ws

	for _, ws := range windowSizes {
		for _, ml := range maxLags {
			if ml > 0 && ws > 4096 {
				continue // Skip large combinations to save time
			}
			for _, nt := range numTracks {
				lagLabel := "SmallLag"
				if ml > 0 {
					lagLabel = "3sLag"
				}
				b.Run(fmt.Sprintf("WindowSize-%d/Lag-%s/Tracks-%d", ws, lagLabel, nt), func(b *testing.B) {
					hopSize := ws / 2
					usedMaxLag := ml
					if usedMaxLag == 0 {
						usedMaxLag = ws
					}
					s, _ := NewSyncer(encoding, channels, ws, hopSize, usedMaxLag, 0, 0)

					// Analyze enough data to fill 2 hop segments
					n := ws + hopSize
					ref := make([]float64, n)
					for i := range ref {
						ref[i] = math.Sin(float64(i) * 0.1)
					}
					comp := make([]float64, n)
					for i := range comp {
						comp[i] = math.Sin(float64(i-10) * 0.1)
					}

					refBytes := float64ToBytes(ref)
					compBytes := float64ToBytes(comp)

					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_ = s.PushReference(ctx, refBytes)
						for tid := range nt {
							_, _ = s.PushComparison(ctx, tid, compBytes)
						}
					}
				})
			}
		}
	}
}

func TestSyncerStream_SignAndStability(t *testing.T) {
	encoding := audio.EncodingPCM{
		PCMFormat:  audio.PCMFormatFloat64LE,
		SampleRate: 44100,
	}
	channels := audio.Channel(1)
	windowSize := 4096
	hopSize := 2048
	maxLag := 8192
	s, _ := NewSyncer(encoding, channels, windowSize, hopSize, maxLag, 0, 0)
	ctx := context.Background()

	// 1. Comparison LAGGING by 100 samples
	// comp[t] = ref[t-100] => shift should be -100
	n := 32768
	ref := make([]float64, n)
	r := rand.New(rand.NewSource(42))
	for i := range ref {
		ref[i] = r.Float64()*2 - 1
	}

	lag := 100
	compLagged := make([]float64, n)
	for i := lag; i < n; i++ {
		compLagged[i] = ref[i-lag]
	}

	_ = s.PushReference(ctx, float64ToBytes(ref))
	results, _ := s.PushComparison(ctx, 0, float64ToBytes(compLagged))

	foundLagged := false
	for _, res := range results {
		fmt.Printf("Lagged result: shift=%v, conf=%v\n", res.Shift, res.Confidence)
		if res.Confidence > 0.15 {
			assert.InDelta(t, float64(-lag), res.Shift, 0.5, "Comparison lagged by 100 should give shift -100")
			foundLagged = true
		}
	}
	assert.True(t, foundLagged, "Should find shift for lagged signal")

	// 2. Comparison LEADING by 100 samples
	// comp[t] = ref[t+100] => shift should be +100
	s, _ = NewSyncer(encoding, channels, windowSize, hopSize, maxLag, 0, 0)

	lead := 100
	compLeading := make([]float64, n)
	for i := 0; i < n-lead; i++ {
		compLeading[i] = ref[i+lead]
	}

	_ = s.PushReference(ctx, float64ToBytes(ref))
	results, _ = s.PushComparison(ctx, 0, float64ToBytes(compLeading))

	foundLeading := false
	for _, res := range results {
		fmt.Printf("Leading result: shift=%v, conf=%v\n", res.Shift, res.Confidence)
		if res.Confidence > 0.15 {
			assert.InDelta(t, float64(lead), res.Shift, 0.5, "Comparison leading by 100 should give shift +100")
			foundLeading = true
		}
	}
	assert.True(t, foundLeading, "Should find shift for leading signal")
}
