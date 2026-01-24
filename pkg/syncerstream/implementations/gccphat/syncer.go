// Package gccphat implements an audio synchronization stream using the GCC-PHAT algorithm.
// GCC-PHAT (Generalized Cross-Correlation with Phase Transform) is a robust method for
// estimating the time-delay between two signals by analyzing their phase differences
// in the frequency domain.
package gccphat

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/mjibson/go-dsp/fft"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/syncer"
	syncergccphat "github.com/xaionaro-go/audio/pkg/syncer/implementations/gccphat"
	"github.com/xaionaro-go/audio/pkg/syncerstream"
)

const (
	defaultWindowDuration = 400 * time.Millisecond
	defaultOverlapFactor  = 0.5

	// Confidence Constants
	// In GCC-PHAT, the whitening process normalizes all frequency bins.
	// For a window of size N, the expected peak magnitude for uncorrelated noise
	// is approximately 1/sqrt(N). We define our thresholds relative to this floor.

	// ThresholdSearchMultiplier is used to determine the minimum confidence
	// required to enter Tracking mode from Search mode. A higher value ensures
	// that we only "lock" on very strong, clear signals.
	ThresholdSearchMultiplier = 10.0

	// ThresholdTrackMultiplier is used to stay in Tracking mode. It is lower
	// than the search multiplier to allow for temporary dips in signal quality
	// without losing the lock.
	ThresholdTrackMultiplier = 5.0
)

// trackState maintains the state for a single comparison signal being synced against the reference.
type trackState struct {
	// compBuffer is a circular buffer for comparison signal samples.
	compBuffer []float64
	// compCount is the total number of samples pushed to this comparison track.
	compCount int64
	// lastAnalysisPos is the global sample index of the last completed analysis window.
	lastAnalysisPos int64
	// fcomp and res are pre-allocated buffers for FFT transformations and results.
	fcomp []complex128
	res   []complex128
	// lastSuccessfulShift is the last verified delay in samples. Used to center the search window.
	lastSuccessfulShift float64
	// isTracking is true when we are in a high-confidence lock on the delay and can use a smaller search window.
	isTracking bool
	// consecutiveHighConf tracks the number of consecutive analysis windows meeting the confidence threshold.
	consecutiveHighConf int
}

// Syncer implements the syncerstream.SyncerStream interface using GCC-PHAT.
type Syncer struct {
	encoding audio.Encoding
	channels audio.Channel
	// windowSize is the number of samples analyzed in each FFT window.
	windowSize int
	// hopSize is the interval between consecutive analysis windows.
	hopSize int
	// maxLag is the maximum searchable delay in either direction.
	maxLag int
	// minFreq and maxFreq are the frequency band limits for cross-correlation.
	minFreq float64
	maxFreq float64
	// refBuffer is a circular buffer for the reference signal samples.
	refBuffer []float64
	// refCount is the total number of samples pushed to the reference stream.
	refCount int64
	// hannWindow is pre-calculated coefficients for the Hann windowing function.
	hannWindow []float64
	// fref is a pre-allocated buffer for the reference signal FFT.
	fref []complex128
	// tracks maps track IDs to their respective trackState.
	tracks map[int]*trackState
	lock   sync.Mutex
}

var _ syncerstream.SyncerStream = (*Syncer)(nil)

type Factory struct {
	WindowSize int
	HopSize    int
	MaxLag     int
	MinFreq    float64
	MaxFreq    float64
}

func (f *Factory) NewSyncer(encoding audio.Encoding, channels audio.Channel) (syncerstream.SyncerStream, error) {
	return NewSyncer(encoding, channels, f.WindowSize, f.HopSize, f.MaxLag, f.MinFreq, f.MaxFreq)
}

// NewSyncer initializes a new GCC-PHAT syncer.
//
// Arguments:
// - encoding: The audio encoding (mandatory, must contain a non-zero sample rate).
// - channels: Number of audio channels (mandatory, must be > 0).
// - windowSize: The size of the snippet to correlate. Defaults to 400ms if <= 0.
// - hopSize: The overlap between windows. Defaults to 50% overlap if <= 0.
// - maxLag: The search range. Defaults to 5 seconds if <= 0.
// - minFreq, maxFreq: Band limiting in Hz. Defaults to 100Hz-12000Hz if both 0.
func NewSyncer(encoding audio.Encoding, channels audio.Channel, windowSize, hopSize, maxLag int, minFreq, maxFreq float64) (*Syncer, error) {
	if encoding == nil {
		return nil, fmt.Errorf("encoding is mandatory")
	}
	if channels <= 0 {
		return nil, fmt.Errorf("channels must be greater than 0: got %d", channels)
	}

	var sampleRate uint32
	if pcm, ok := encoding.(audio.EncodingPCM); ok {
		sampleRate = uint32(pcm.SampleRate)
	}
	if sampleRate == 0 {
		return nil, fmt.Errorf("sample rate is mandatory and could not be determined from encoding %T", encoding)
	}

	if windowSize <= 0 {
		windowSize = 1
		for windowSize < int(float64(sampleRate)*defaultWindowDuration.Seconds()) {
			windowSize <<= 1
		}
	}
	if hopSize <= 0 {
		hopSize = int(float64(windowSize) * defaultOverlapFactor)
	}
	if maxLag <= 0 {
		maxLag = int(sampleRate * 5)
	}

	// Default band limiting: 100Hz to 12000Hz is a good general range for audio sync
	if minFreq == 0 && maxFreq == 0 {
		minFreq = 100
		maxFreq = 12000
	}

	hann := make([]float64, windowSize)
	for i := 0; i < windowSize; i++ {
		hann[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(windowSize-1)))
	}
	bufferSize := (maxLag + windowSize) * 4
	s := &Syncer{
		encoding:   encoding,
		channels:   channels,
		windowSize: windowSize,
		hopSize:    hopSize,
		maxLag:     maxLag,
		minFreq:    minFreq,
		maxFreq:    maxFreq,
		refBuffer:  make([]float64, bufferSize),
		hannWindow: hann,
		tracks:     make(map[int]*trackState),
	}
	// Initial fref is sized for full Search mode
	n := s.getOptimalFFTSize(maxLag)
	s.fref = make([]complex128, n)
	return s, nil
}

func (s *Syncer) getOptimalFFTSize(maxLag int) int {
	n := 1
	for n < s.windowSize+maxLag {
		n <<= 1
	}
	return n << 1 // 2x padding for linear correlation
}

func (s *Syncer) getTrackState(trackID int) *trackState {
	s.lock.Lock()
	defer s.lock.Unlock()
	ts, ok := s.tracks[trackID]
	if !ok {
		n := len(s.fref) // Start with Search-sized buffers
		ts = &trackState{
			compBuffer:      make([]float64, len(s.refBuffer)),
			fcomp:           make([]complex128, n),
			res:             make([]complex128, n),
			lastAnalysisPos: -int64(s.hopSize),
		}
		s.tracks[trackID] = ts
	}
	return ts
}

func (s *Syncer) PushReference(ctx context.Context, data []byte) error {
	samples, err := syncergccphat.ToSamples(s.encoding, s.channels, data)
	if err != nil {
		return err
	}
	for _, v := range samples {
		s.refBuffer[s.refCount%int64(len(s.refBuffer))] = v
		s.refCount++
	}
	return nil
}

// PushComparison processes comparison stream data and returns detected shifts.
//
// It uses an adaptive windowing strategy:
//  1. Initially, it searches a large range (maxLag) to find a rough match.
//  2. Once a match with sufficient confidence is found repeatedly, it enters "Track" mode.
//  3. In "Track" mode, it narrows the search window to windowSize near the last known shift,
//     significantly reducing CPU usage while maintaining sync.
func (s *Syncer) PushComparison(ctx context.Context, trackID int, data []byte) ([]syncer.ShiftResult, error) {
	samples, err := syncergccphat.ToSamples(s.encoding, s.channels, data)
	if err != nil {
		return nil, err
	}
	ts := s.getTrackState(trackID)
	for _, v := range samples {
		ts.compBuffer[ts.compCount%int64(len(ts.compBuffer))] = v
		ts.compCount++
	}
	var results []syncer.ShiftResult
	for {
		nextPos := ts.lastAnalysisPos + int64(s.hopSize)
		if nextPos+int64(s.windowSize) > ts.compCount {
			break
		}

		maxLag := s.maxLag
		searchStart := nextPos + int64(ts.lastSuccessfulShift)
		if !ts.isTracking {
			searchStart = nextPos
		} else {
			maxLag = s.windowSize
		}

		if searchStart < 0 {
			searchStart = 0
		}
		if searchStart+int64(s.windowSize) > s.refCount {
			break
		}

		shift, confidence, activeBins, err := s.analyze(ts, nextPos, searchStart, maxLag)
		if err != nil {
			return results, err
		}

		searchOrigin := searchStart - int64(maxLag)
		totalShift := float64(searchOrigin-nextPos) + shift

		// Confidence Heuristic:
		// In GCC-PHAT, the whitening process normalizes all frequency bins.
		// For a window of size N, the expected peak magnitude for uncorrelated noise
		// is approximately 1/sqrt(N). We define our thresholds relative to this floor.
		noiseFloor := 1.0 / math.Sqrt(float64(activeBins))
		thresholdSearch := noiseFloor * ThresholdSearchMultiplier
		thresholdTrack := noiseFloor * ThresholdTrackMultiplier

		if confidence > thresholdSearch {
			ts.lastSuccessfulShift = totalShift
			ts.consecutiveHighConf++
			// We need a few consecutive high-confidence hits to consider we've "locked" onto the signal.
			if ts.consecutiveHighConf >= 2 {
				ts.isTracking = true
			}
		} else if ts.isTracking && confidence > thresholdTrack {
			// If we are already tracking, we can accept a slightly lower confidence to maintain the lock,
			// which helps in noisy environments where the peak might temporarily dip.
			ts.lastSuccessfulShift = totalShift
			ts.consecutiveHighConf++
		} else {
			// If confidence falls below our tracking threshold, we revert to full-range search.
			if confidence < thresholdTrack {
				ts.isTracking = false
				ts.consecutiveHighConf = 0
			}
		}

		results = append(results, syncer.ShiftResult{
			SampleOffset: nextPos,
			Shift:        totalShift,
			Confidence:   confidence,
		})
		ts.lastAnalysisPos = nextPos
	}
	return results, nil
}

// analyze performs a single GCC-PHAT correlation between a window in the comparison signal
// and a search range in the reference signal.
//
// searchStart is the global sample position in the reference stream where we expect
// to find the match. maxLag defines how far in both directions we should search.
//
// The reference signal snippet is extracted starting from searchStart - maxLag,
// creating a search space of windowSize + 2 * maxLag samples.
func (s *Syncer) analyze(ts *trackState, pos int64, searchStart int64, maxLag int) (float64, float64, int, error) {
	// searchOrigin is the start of the reference buffer snippet we extract.
	// Centering the search around searchStart allows us to detect shifts in both directions.
	searchOrigin := searchStart - int64(maxLag)
	searchSamples := s.windowSize + 2*maxLag

	// Determine optimal FFT size for this specific maxLag.
	// We need 2^n >= windowSize + searchSamples - 1 to avoid circular convolution.
	// searchSamples is windowSize + 2*maxLag.
	// So 2^n >= 2*windowSize + 2*maxLag - 1.
	targetN := 2*s.windowSize + 2*maxLag - 1
	n := 1
	for n < targetN {
		n <<= 1
	}

	if n > len(s.fref) {
		n = len(s.fref)
	}

	fref := s.fref[:n]
	fcomp := ts.fcomp[:n]

	for i := 0; i < n; i++ {
		fref[i] = 0
		fcomp[i] = 0
	}

	// Prepare comparison snippet with Hann windowing
	compLen := int64(len(ts.compBuffer))
	nonZeroComp := 0
	for i := 0; i < s.windowSize; i++ {
		v := ts.compBuffer[(pos+int64(i))%compLen]
		if v != 0 {
			nonZeroComp++
		}
		fcomp[i] = complex(v*s.hannWindow[i], 0)
	}

	// Prepare reference snippet
	refLen := int64(len(s.refBuffer))
	nonZeroRef := 0
	for i := 0; i < searchSamples && i < n; i++ {
		globalIdx := searchOrigin + int64(i)
		if globalIdx >= 0 && globalIdx >= s.refCount-refLen && globalIdx < s.refCount {
			actualIdx := globalIdx % refLen
			if actualIdx < 0 {
				actualIdx += refLen
			}
			v := s.refBuffer[actualIdx]
			if v != 0 {
				nonZeroRef++
			}
			fref[i] = complex(v, 0)
		} else {
			fref[i] = 0
		}
	}

	ffref := fft.FFT(fref)
	ffcomp := fft.FFT(fcomp)

	// Cross-Correlate with Frequency Band Limiting
	pcm, ok := s.encoding.(audio.EncodingPCM)
	if !ok || pcm.SampleRate == 0 {
		return 0, 0, 0, fmt.Errorf("sample rate is required for band limiting")
	}
	sampleRate := float64(pcm.SampleRate)
	shift, confidence, err := syncergccphat.CrossCorrelate(ffref, ffcomp, sampleRate, s.minFreq, s.maxFreq)
	if err != nil {
		return 0, 0, 0, err
	}

	// Calculate active bins (the same used inside CrossCorrelate)
	binMin := int(s.minFreq * float64(n) / sampleRate)
	binMax := n / 2
	if s.maxFreq > 0 && s.maxFreq < sampleRate/2 {
		binMax = int(s.maxFreq * float64(n) / sampleRate)
	}
	activeBins := 2 * (binMax - binMin)
	if activeBins <= 0 {
		activeBins = 1
	}

	return shift, confidence, activeBins, nil
}

func (s *Syncer) Encoding(_ context.Context) (audio.Encoding, error) { return s.encoding, nil }
func (s *Syncer) Channels(_ context.Context) (audio.Channel, error)  { return s.channels, nil }
func (s *Syncer) Close() error                                       { return nil }
