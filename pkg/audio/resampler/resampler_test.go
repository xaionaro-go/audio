package resampler

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

func TestResampler(t *testing.T) {
	t.Run("Identity_S16LE_Mono_44100", func(t *testing.T) {
		inFmt := Format{
			Channels:   1,
			SampleRate: 44100,
			PCMFormat:  types.PCMFormatS16LE,
		}
		// S16 is 2 bytes per sample. 100 samples = 200 bytes.
		data := make([]byte, 200)
		for i := 0; i < 100; i++ {
			binary.LittleEndian.PutUint16(data[i*2:], uint16(i*100))
		}
		reader := bytes.NewReader(data)
		r, err := NewResampler(inFmt, reader, inFmt)
		require.NoError(t, err)

		out := make([]byte, 200)
		n, err := r.Read(out)
		assert.NoError(t, err)
		assert.Equal(t, 200, n)
		assert.Equal(t, data, out)
	})

	t.Run("Conversion_U8_to_Float32LE_Mono", func(t *testing.T) {
		inFmt := Format{
			Channels:   1,
			SampleRate: 44100,
			PCMFormat:  types.PCMFormatU8,
		}
		outFmt := Format{
			Channels:   1,
			SampleRate: 44100,
			PCMFormat:  types.PCMFormatFloat32LE,
		}
		// 128 in U8 is approx 0.0 in Float32
		data := []byte{0, 128, 255}
		reader := bytes.NewReader(data)
		r, err := NewResampler(inFmt, reader, outFmt)
		require.NoError(t, err)

		out := make([]byte, 3*4)
		n, err := r.Read(out)
		assert.NoError(t, err)
		assert.Equal(t, 12, n)

		v0 := math.Float32frombits(binary.LittleEndian.Uint32(out[0:4]))
		v1 := math.Float32frombits(binary.LittleEndian.Uint32(out[4:8]))
		v2 := math.Float32frombits(binary.LittleEndian.Uint32(out[8:12]))

		assert.InDelta(t, -1.0, v0, 0.01)
		assert.InDelta(t, 0.0, v1, 0.01)
		assert.InDelta(t, 1.0, v2, 0.01)
	})

	t.Run("Resampling_44100_to_22050", func(t *testing.T) {
		inFmt := Format{
			Channels:   1,
			SampleRate: 44100,
			PCMFormat:  types.PCMFormatU8,
		}
		outFmt := Format{
			Channels:   1,
			SampleRate: 22050,
			PCMFormat:  types.PCMFormatU8,
		}
		data := make([]byte, 100)
		for i := range data {
			data[i] = byte(i)
		}
		reader := bytes.NewReader(data)
		r, err := NewResampler(inFmt, reader, outFmt)
		require.NoError(t, err)

		out := make([]byte, 50)
		n, err := r.Read(out)
		assert.NoError(t, err)
		assert.Equal(t, 50, n)
		// Basic check: should take roughly every second sample
		assert.Equal(t, data[0], out[0])
		assert.Equal(t, data[2], out[1])
	})

	t.Run("Channels_Mono_to_Stereo", func(t *testing.T) {
		inFmt := Format{
			Channels:   1,
			SampleRate: 44100,
			PCMFormat:  types.PCMFormatU8,
		}
		outFmt := Format{
			Channels:   2,
			SampleRate: 44100,
			PCMFormat:  types.PCMFormatU8,
		}
		data := []byte{10, 20, 30}
		reader := bytes.NewReader(data)
		r, err := NewResampler(inFmt, reader, outFmt)
		require.NoError(t, err)

		out := make([]byte, 6)
		n, err := r.Read(out)
		assert.NoError(t, err)
		assert.Equal(t, 6, n)
		assert.Equal(t, []byte{10, 10, 20, 20, 30, 30}, out)
	})

	t.Run("Channels_Stereo_to_Mono", func(t *testing.T) {
		inFmt := Format{
			Channels:   2,
			SampleRate: 44100,
			PCMFormat:  types.PCMFormatU8,
		}
		outFmt := Format{
			Channels:   1,
			SampleRate: 44100,
			PCMFormat:  types.PCMFormatU8,
		}
		data := []byte{100, 200, 50, 150}
		reader := bytes.NewReader(data)
		r, err := NewResampler(inFmt, reader, outFmt)
		require.NoError(t, err)

		out := make([]byte, 2)
		n, err := r.Read(out)
		assert.NoError(t, err)
		assert.Equal(t, 2, n)
		// (100+200)/2 = 150 -> approx (scaled back to U8)
		assert.Equal(t, byte(150), out[0])
		assert.Equal(t, byte(100), out[1]) // (50+150)/2 = 100
	})
}
