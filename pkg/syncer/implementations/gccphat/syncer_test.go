package gccphat

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xaionaro-go/audio/pkg/audio"
)

func float32ToBytes(data []float32) []byte {
	res := make([]byte, len(data)*4)
	for i, v := range data {
		binary.LittleEndian.PutUint32(res[i*4:], math.Float32bits(v))
	}
	return res
}

func TestSyncer_CalculateShiftBetween(t *testing.T) {
	encoding := audio.EncodingPCM{
		PCMFormat:  audio.PCMFormatFloat32LE,
		SampleRate: 44100,
	}
	channels := audio.Channel(1)
	s, err := NewSyncer(encoding, channels)
	assert.NoError(t, err)

	t.Run("ahead by 10", func(t *testing.T) {
		ref := make([]float32, 1000)
		ref[500] = 1.0

		comp := make([]float32, 1000)
		comp[490] = 1.0 // comp is ahead by 10 samples (event at 490 vs 500)

		results, err := s.CalculateShiftBetween(context.Background(), float32ToBytes(ref), float32ToBytes(comp))
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.InDelta(t, 10.0, results[0].Shift, 0.5)
		assert.Greater(t, results[0].Confidence, 0.4)
	})

	t.Run("delayed by 10 (ahead by -10)", func(t *testing.T) {
		ref := make([]float32, 1000)
		ref[500] = 1.0

		comp := make([]float32, 1000)
		comp[510] = 1.0 // comp is delayed by 10 samples

		results, err := s.CalculateShiftBetween(context.Background(), float32ToBytes(ref), float32ToBytes(comp))
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.InDelta(t, -10.0, results[0].Shift, 0.5)
		assert.Greater(t, results[0].Confidence, 0.4)
	})

	t.Run("no shift", func(t *testing.T) {
		ref := make([]float32, 1000)
		ref[500] = 1.0

		comp := make([]float32, 1000)
		comp[500] = 1.0

		results, err := s.CalculateShiftBetween(context.Background(), float32ToBytes(ref), float32ToBytes(comp))
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.InDelta(t, 0.0, results[0].Shift, 0.5)
		assert.Greater(t, results[0].Confidence, 0.4)
	})

	t.Run("complex signal ahead by 5", func(t *testing.T) {
		ref := make([]float32, 2000)
		for i := range ref {
			ref[i] = float32(math.Sin(float64(i) * 0.1))
		}

		comp := make([]float32, 2000)
		copy(comp, ref[5:]) // comp[0] = ref[5], so it's ahead by 5

		results, err := s.CalculateShiftBetween(context.Background(), float32ToBytes(ref), float32ToBytes(comp))
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.InDelta(t, 5.0, results[0].Shift, 0.5)
		assert.Greater(t, results[0].Confidence, 0.4)
	})
}

func BenchmarkSyncer_CalculateShiftBetween(b *testing.B) {
	encoding := audio.EncodingPCM{
		PCMFormat:  audio.PCMFormatFloat32LE,
		SampleRate: 44100,
	}
	channels := audio.Channel(1)
	s, _ := NewSyncer(encoding, channels)
	ctx := context.Background()

	sizes := []int{1000, 10000, 100000}
	for _, n := range sizes {
		b.Run(fmt.Sprintf("size-%d", n), func(b *testing.B) {
			ref := make([]float32, n)
			for i := range ref {
				ref[i] = float32(math.Sin(float64(i) * 0.1))
			}
			comp := make([]float32, n)
			copy(comp, ref[n/10:])

			refBytes := float32ToBytes(ref)
			compBytes := float32ToBytes(comp)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := s.CalculateShiftBetween(ctx, refBytes, compBytes)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
