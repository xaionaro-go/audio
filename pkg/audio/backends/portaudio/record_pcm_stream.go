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

const (
	RecordBufferSize = time.Millisecond * 100
)

type RecordPCMStream struct {
	PortAudioStream  *portaudio.Stream
	InputBuffer      []byte
	OutputBuffer     []byte
	Writer           io.Writer
	CancelFunc       context.CancelFunc
	WaitGroup        sync.WaitGroup
	StartWritingChan chan struct{}
	StartReadingChan chan struct{}
}

func newRecordPCMStream[T any](
	ctx context.Context,
	sampleRate types.SampleRate,
	channels types.Channel,
) (*RecordPCMStream, error) {
	bufferItemsCount := int(RecordBufferSize.Seconds() * float64(sampleRate))

	var sample T
	buf := make([]T, bufferItemsCount)
	logger.Debugf(ctx, "newRecordPCMStream: %T, %d, %d %s(%d)", sample, sampleRate, channels, RecordBufferSize, bufferItemsCount)
	logger.Debugf(ctx, "input buffer: %T (size: %d)", buf, len(buf))
	stream, err := portaudio.OpenDefaultStream(int(channels), 0, float64(sampleRate), bufferItemsCount, buf)
	if err != nil {
		return nil, err
	}

	ptr := unsafe.SliceData(buf)
	bytesBuf := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(buf)*int(unsafe.Sizeof(sample)))

	logger.Debugf(ctx, "input bytes buffer size: %d", len(bytesBuf))
	s := &RecordPCMStream{
		PortAudioStream:  stream,
		InputBuffer:      bytesBuf,
		OutputBuffer:     make([]byte, len(bytesBuf)),
		StartWritingChan: make(chan struct{}),
		StartReadingChan: make(chan struct{}),
	}
	return s, nil
}

func (s *RecordPCMStream) init(
	ctx context.Context,
	writer io.Writer,
) error {
	s.Writer = writer
	ctx, s.CancelFunc = context.WithCancel(ctx)

	err := s.PortAudioStream.Start()
	if err != nil {
		return fmt.Errorf("unable to start the stream: %w", err)
	}

	s.WaitGroup.Add(1)
	observability.Go(ctx, func(ctx context.Context) {
		defer s.WaitGroup.Done()
		<-ctx.Done()
		s.Close()
	})
	s.WaitGroup.Add(1)
	observability.Go(ctx, func(ctx context.Context) {
		defer s.WaitGroup.Done()
		defer s.CancelFunc()
		s.readerLoop(ctx)
	})
	s.WaitGroup.Add(1)
	observability.Go(ctx, func(ctx context.Context) {
		defer s.WaitGroup.Done()
		defer s.CancelFunc()
		s.writerLoop(ctx)
	})
	return nil
}

func (s *RecordPCMStream) readerLoop(
	ctx context.Context,
) (_ret error) {
	logger.Debugf(ctx, "readerLoop")
	defer func() { logger.Debugf(ctx, "/readerLoop: %v", _ret) }()
	defer func() {
		close(s.StartWritingChan)
	}()

	for {
		logger.Tracef(ctx, "Read")
		err := s.PortAudioStream.Read()
		logger.Tracef(ctx, "/Read: %v", err)
		if err != nil {
			return fmt.Errorf("unable to read: %w", err)
		}
		select {
		case s.StartWritingChan <- struct{}{}:
		case <-s.StartReadingChan:
			return
		}
		<-s.StartReadingChan
	}
}

func (s *RecordPCMStream) writerLoop(
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
		n, err := s.Writer.Write(s.OutputBuffer)
		logger.Tracef(ctx, "/Write: %d %v", n, err)
		if n != len(s.OutputBuffer) {
			return fmt.Errorf("invalid write length: %d != %d", n, len(s.OutputBuffer))
		}
	}
}

func (s *RecordPCMStream) Close() error {
	s.CancelFunc()
	return s.PortAudioStream.Abort()
}
func (s *RecordPCMStream) Drain() error {
	s.WaitGroup.Wait()
	return nil
}
