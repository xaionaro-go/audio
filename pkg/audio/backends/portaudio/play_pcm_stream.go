package portaudio

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
	"unsafe"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/gordonklaus/portaudio"
	"github.com/xaionaro-go/audio/pkg/audio/types"
	"github.com/xaionaro-go/observability"
)

type PlayPCMStream struct {
	PortAudioStream  *portaudio.Stream
	OutputBuffer     []byte
	InputBuffer      []byte
	Reader           io.Reader
	CancelFunc       context.CancelFunc
	WaitGroup        sync.WaitGroup
	StartWritingChan chan struct{}
	StartReadingChan chan struct{}
}

func newPlayPCMStream[T any](
	ctx context.Context,
	sampleRate types.SampleRate,
	channels types.Channel,
	bufferSize time.Duration,
) (*PlayPCMStream, error) {
	bufferItemsCount := int(bufferSize.Seconds() * float64(sampleRate))

	var sample T
	buf := make([]T, bufferItemsCount)
	logger.Debugf(ctx, "newPlayPCMStream: %T, %d, %d %s(%d)", sample, sampleRate, channels, bufferSize, bufferItemsCount)
	logger.Debugf(ctx, "output buffer: %T (size: %d)", buf, len(buf))
	stream, err := portaudio.OpenDefaultStream(0, int(channels), float64(sampleRate), bufferItemsCount, &buf)
	if err != nil {
		return nil, err
	}

	ptr := unsafe.SliceData(buf)
	bytesBuf := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(buf)*int(unsafe.Sizeof(sample)))

	logger.Debugf(ctx, "output bytes buffer size: %d", len(bytesBuf))
	s := &PlayPCMStream{
		PortAudioStream:  stream,
		OutputBuffer:     bytesBuf,
		InputBuffer:      make([]byte, len(bytesBuf)),
		StartWritingChan: make(chan struct{}),
		StartReadingChan: make(chan struct{}),
	}
	return s, nil
}

func (s *PlayPCMStream) init(
	ctx context.Context,
	rawReader io.Reader,
) error {
	s.Reader = rawReader
	ctx, s.CancelFunc = context.WithCancel(ctx)

	err := s.PortAudioStream.Start()
	if err != nil {
		return fmt.Errorf("unable to start the stream: %w", err)
	}

	s.WaitGroup.Add(1)
	observability.Go(ctx, func() {
		defer s.WaitGroup.Done()
		<-ctx.Done()
		s.Close()
	})
	s.WaitGroup.Add(1)
	observability.Go(ctx, func() {
		defer s.WaitGroup.Done()
		defer s.CancelFunc()
		s.readerLoop(ctx)
	})
	s.WaitGroup.Add(1)
	observability.Go(ctx, func() {
		defer s.WaitGroup.Done()
		defer s.CancelFunc()
		s.writerLoop(ctx)
	})
	return nil
}

func (s *PlayPCMStream) readerLoop(
	ctx context.Context,
) (_ret error) {
	logger.Debugf(ctx, "readerLoop")
	defer func() { logger.Debugf(ctx, "/readerLoop: %v", _ret) }()
	defer func() {
		close(s.StartWritingChan)
	}()

	for {
		buf := s.InputBuffer
		for cap(buf) > 0 {
			logger.Tracef(ctx, "Read")
			n, err := s.Reader.Read(buf)
			logger.Tracef(ctx, "/Read: %v %v", n, err)
			if err != nil {
				return fmt.Errorf("unable to read: %w", err)
			}
			buf = buf[n:]
			logger.Tracef(ctx, "left to read: %d", cap(buf))
		}
		select {
		case s.StartWritingChan <- struct{}{}:
		case <-s.StartReadingChan:
			return
		}
		<-s.StartReadingChan
	}
}

func (s *PlayPCMStream) writerLoop(
	ctx context.Context,
) (_ret error) {
	logger.Debugf(ctx, "writerLoop")
	defer func() { logger.Debugf(ctx, "/writerLoop: %v", _ret) }()
	defer func() {
		close(s.StartReadingChan)
	}()

	for {
		<-s.StartWritingChan
		copy(s.OutputBuffer, s.InputBuffer)
		s.StartReadingChan <- struct{}{}

		logger.Tracef(ctx, "Write")
		err := s.PortAudioStream.Write()
		logger.Tracef(ctx, "/Write: %v", err)
		if err != nil {
			return fmt.Errorf("unable to write: %w", err)
		}
	}
}

func (s *PlayPCMStream) Close() error {
	s.CancelFunc()
	return s.PortAudioStream.Abort()
}
func (s *PlayPCMStream) Drain() error {
	s.WaitGroup.Wait()
	return nil
}
