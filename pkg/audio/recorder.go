package audio

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/hashicorp/go-multierror"
	"github.com/xaionaro-go/audio/pkg/audio/registry"
)

type Recorder struct {
	RecorderPCM
}

func NewRecorder(recorderPCM RecorderPCM) *Recorder {
	return &Recorder{
		RecorderPCM: recorderPCM,
	}
}

var (
	lastSuccessfulRecorderFactory       registry.RecorderPCMFactory
	lastSuccessfulRecorderFactoryLocker sync.Mutex
)

func getLastSuccessfulRecorderFactory() registry.RecorderPCMFactory {
	lastSuccessfulRecorderFactoryLocker.Lock()
	defer lastSuccessfulRecorderFactoryLocker.Unlock()
	return lastSuccessfulRecorderFactory
}

func NewRecorderAuto(
	ctx context.Context,
) *Recorder {
	factory := getLastSuccessfulRecorderFactory()
	if factory != nil {
		recorder, err := factory.NewRecorderPCM()
		if err == nil {
			if err := recorder.Ping(ctx); err == nil {
				return NewRecorder(recorder)
			}
		}
	}

	var mErr *multierror.Error
	for _, factory := range registry.RecorderFactories() {
		recorder, err := factory.NewRecorderPCM()
		logger.Debugf(ctx, "initializing recorder %T result is %v", recorder, err)
		if err != nil {
			mErr = multierror.Append(mErr, fmt.Errorf("unable to initialize %T: %w", recorder, err))
			continue
		}

		err = recorder.Ping(ctx)
		logger.Debugf(ctx, "pinging PCM recorder %T result is %v", recorder, err)
		if err != nil {
			mErr = multierror.Append(mErr, fmt.Errorf("unable to ping %T: %w", recorder, err))
			continue
		}

		lastSuccessfulRecorderFactoryLocker.Lock()
		defer lastSuccessfulRecorderFactoryLocker.Unlock()
		lastSuccessfulRecorderFactory = factory
		return NewRecorder(recorder)
	}

	logger.Infof(ctx, "was unable to initialize any PCM recorder")
	return &Recorder{
		RecorderPCM: RecorderPCMDummy{},
	}
}

func (a *Recorder) RecordPCM(
	ctx context.Context,
	sampleRate SampleRate,
	channels Channel,
	pcmFormat PCMFormat,
	pcmWriter io.Writer,
) (RecordStream, error) {
	return a.RecorderPCM.RecordPCM(
		ctx,
		sampleRate,
		channels,
		pcmFormat,
		pcmWriter,
	)
}
