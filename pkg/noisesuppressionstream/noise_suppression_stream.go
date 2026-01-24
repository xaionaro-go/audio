package noisesuppressionstream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/iamcalledrob/circular"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/noisesuppression"
	"github.com/xaionaro-go/observability"
)

const (
	debugBypassNoiseSuppression = false
)

type NoiseSuppressionStream struct {
	noisesuppression.NoiseSuppression
	encoding           audio.Encoding
	channels           audio.Channel
	inputBufferLocker  sync.Mutex
	inputBuffer        *circular.Buffer
	outputBufferLocker sync.Mutex
	outputBuffer       *circular.Buffer
	resultError        error
	readCtx            context.Context

	readProgressedCh                   chan struct{}
	noiseSuppressionInputProgressedCh  chan struct{}
	noiseSuppressionOutputProgressedCh chan struct{}
	outputProgressedCh                 chan struct{}
}

var _ io.Reader = (*NoiseSuppressionStream)(nil)

func NewNoiseSuppressionStream(
	ctx context.Context,
	input io.Reader,
	noiseSuppression noisesuppression.NoiseSuppression,
	inputBufferSize uint,
	outputBufferSize uint,
) (*NoiseSuppressionStream, error) {
	encoding, err := noiseSuppression.Encoding(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to get the encoding of the noise suppression: %w", err)
	}
	channels, err := noiseSuppression.Channels(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to get the amount of channels of the noise suppression: %w", err)
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	s := &NoiseSuppressionStream{
		NoiseSuppression: noiseSuppression,
		encoding:         encoding,
		channels:         channels,
		inputBuffer:      circular.NewBuffer(int(inputBufferSize)),
		outputBuffer:     circular.NewBuffer(int(outputBufferSize)),
		readCtx:          ctx,

		readProgressedCh:                   make(chan struct{}),
		noiseSuppressionInputProgressedCh:  make(chan struct{}),
		noiseSuppressionOutputProgressedCh: make(chan struct{}),
		outputProgressedCh:                 make(chan struct{}),
	}
	observability.Go(ctx, func(ctx context.Context) {
		defer cancelFunc()
		err := s.readerLoop(ctx, input)
		s.inputBufferLocker.Lock()
		defer s.inputBufferLocker.Unlock()
		if err != nil && s.resultError == nil {
			s.resultError = fmt.Errorf("got an error from the reader loop: %w", err)
		}
	})
	observability.Go(ctx, func(ctx context.Context) {
		defer cancelFunc()
		err = s.noiseSuppressionLoop(ctx)
		s.inputBufferLocker.Lock()
		defer s.inputBufferLocker.Unlock()
		if err != nil && s.resultError == nil {
			s.resultError = fmt.Errorf("got an error from the noise suppressor loop: %w", err)
		}
	})
	return s, nil
}

func (s *NoiseSuppressionStream) readerLoop(
	ctx context.Context,
	input io.Reader,
) (_err error) {
	logger.Tracef(ctx, "readerLoop")
	defer func() { logger.Tracef(ctx, "/readerLoop %v", _err) }()

	readBuf := make([]byte, 65536)
	shortestMessageSize := s.encoding.BytesPerSample() * uint(s.channels)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		logger.Tracef(ctx, "readerLoop: Read()")
		n, err := input.Read(readBuf)
		logger.Tracef(ctx, "/readerLoop: Read(): %v %v", n, err)
		if err != nil {
			return fmt.Errorf("unable to read the backend: %w", err)
		}
		if n < 0 {
			return fmt.Errorf("received invalid value of received bytes: %d", n)
		}
		if n%int(shortestMessageSize) != 0 {
			return fmt.Errorf("received a message of size %d that is not multiple of %d*%d", shortestMessageSize, s.encoding.BytesPerSample(), uint(s.channels))
		}

		if err := func() error {
			s.inputBufferLocker.Lock()
			defer s.inputBufferLocker.Unlock()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				w, err := s.inputBuffer.Write(readBuf[:n])
				if err != nil {
					if errors.Is(err, circular.ErrNoSpace) {
						s.waitForNoiseSuppressionInputProgressed(ctx)
						continue
					}
					return fmt.Errorf("unable to write to the circular buffer: %w", err)
				}
				if w != n {
					return fmt.Errorf("wrote != read: %d != %d", w, n)
				}
				break
			}
			logger.Tracef(ctx, "closing readProgressedCh")
			oldCh := s.readProgressedCh
			s.readProgressedCh = make(chan struct{})
			close(oldCh)
			return nil
		}(); err != nil {
			return err
		}
	}
}

func (s *NoiseSuppressionStream) waitForNoiseSuppressionInputProgressed(ctx context.Context) {
	logger.Tracef(ctx, "waitForNoiseSuppressionInputProgressed")
	defer logger.Tracef(ctx, "/waitForNoiseSuppressionInputProgressed")

	ch := s.noiseSuppressionInputProgressedCh
	s.inputBufferLocker.Unlock()
	defer s.inputBufferLocker.Lock()
	select {
	case <-ctx.Done():
	case <-ch:
		logger.Tracef(ctx, "waitForNoiseSuppressionInputProgressed: received an event")
	}
}

func (s *NoiseSuppressionStream) noiseSuppressionLoop(ctx context.Context) (_err error) {
	logger.Tracef(ctx, "noiseSuppressionLoop")
	defer func() { logger.Tracef(ctx, "/noiseSuppressionLoop: %v", _err) }()

	frameSize := s.ChunkSize()
	logger.Debugf(ctx, "frameSize: %d", frameSize)

	inputBuf := make([]byte, frameSize)
	outputBuf := make([]byte, frameSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		receivedCount := 0
		for {
			var waitCh chan struct{}
			if err := func() error {
				var oldCh chan struct{}
				s.inputBufferLocker.Lock()
				defer s.inputBufferLocker.Unlock()
				n, err := s.inputBuffer.Read(inputBuf[receivedCount:])
				waitCh = s.readProgressedCh
				if err != nil && !errors.Is(err, io.EOF) {
					return fmt.Errorf("unable to read from the circular buffer: %w", err)
				}
				if n < 0 {
					return fmt.Errorf("received a negative count: %d", n)
				}
				receivedCount += n
				logger.Tracef(ctx, "closing noiseSuppressionInputProgressedCh")
				oldCh, s.noiseSuppressionInputProgressedCh = s.noiseSuppressionInputProgressedCh, make(chan struct{})
				close(oldCh)
				return nil
			}(); err != nil {
				return err
			}
			if receivedCount >= int(frameSize) {
				break
			}
			select {
			case <-ctx.Done():
			case <-waitCh:
				logger.Tracef(ctx, "noiseSuppressionLoop: received a read event")
			}
		}

		if debugBypassNoiseSuppression {
			time.Sleep(time.Millisecond)
			copy(outputBuf, inputBuf)
		} else {
			logger.Tracef(ctx, "s.NoiseSuppression.SuppressNoise")
			_, err := s.NoiseSuppression.SuppressNoise(ctx, inputBuf, outputBuf)
			logger.Tracef(ctx, "/s.NoiseSuppression.SuppressNoise: %v", err)
			if err != nil {
				return fmt.Errorf("unable to noise-suppress: %w", err)
			}
		}

		if err := func() error {
			logger.Tracef(ctx, "s.outputBufferLocker.Lock()")
			s.outputBufferLocker.Lock()
			defer s.outputBufferLocker.Unlock()
			logger.Tracef(ctx, "/s.outputBufferLocker.Lock()")

			w, err := s.outputBuffer.Write(outputBuf)
			if err != nil {
				if errors.Is(err, circular.ErrNoSpace) {
					s.waitForOutput(ctx)
					return nil
				}
				return fmt.Errorf("unable to write to the circular buffer: %w", err)
			}
			if w != len(outputBuf) {
				return fmt.Errorf("wrote != read: %d != %d", w, len(outputBuf))
			}
			logger.Tracef(ctx, "closing noiseSuppressionOutputProgressedCh")
			var oldCh chan struct{}
			oldCh, s.noiseSuppressionOutputProgressedCh = s.noiseSuppressionOutputProgressedCh, make(chan struct{})
			close(oldCh)
			return nil
		}(); err != nil {
			return err
		}
	}
}

func (s *NoiseSuppressionStream) waitForOutput(ctx context.Context) {
	logger.Tracef(ctx, "waitForOutput")
	defer logger.Tracef(ctx, "/waitForOutput")

	ch := s.outputProgressedCh
	s.outputBufferLocker.Unlock()
	defer s.outputBufferLocker.Lock()
	select {
	case <-ctx.Done():
	case <-ch:
		logger.Tracef(ctx, "waitForOutput: received an event")
	}
}

func (s *NoiseSuppressionStream) Read(pcm []byte) (_ret int, _err error) {
	logger.Tracef(s.readCtx, "Read, len:%d", len(pcm))
	defer func() { logger.Tracef(s.readCtx, "/Read, len:%d: %d, %v", len(pcm), _ret, _err) }()

	s.outputBufferLocker.Lock()
	defer s.outputBufferLocker.Unlock()
	if s.resultError != nil {
		return 0, s.resultError
	}

	for {
		logger.Tracef(s.readCtx, "Read: s.outputBuffer.Read()")
		n, err := s.outputBuffer.Read(pcm)
		logger.Tracef(s.readCtx, "/Read: s.outputBuffer.Read(): %v %v", n, err)
		if err == nil {
			return n, nil
		}
		if !errors.Is(err, io.EOF) {
			return n, err
		}
		s.waitForNoiseSuppressionOutputProgressed(s.readCtx)
	}
}

func (s *NoiseSuppressionStream) waitForNoiseSuppressionOutputProgressed(ctx context.Context) {
	logger.Tracef(ctx, "waitForNoiseSuppressionOutputProgressed")
	defer logger.Tracef(ctx, "/waitForNoiseSuppressionOutputProgressed")

	ch := s.noiseSuppressionOutputProgressedCh
	s.outputBufferLocker.Unlock()
	defer s.outputBufferLocker.Lock()
	select {
	case <-ctx.Done():
	case <-ch:
		logger.Tracef(ctx, "waitForNoiseSuppressionOutputProgressed: received an event")
	}
}
