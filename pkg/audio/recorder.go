package audio

import (
	"context"
	"io"
	"sync"

	"github.com/facebookincubator/go-belt/tool/logger"
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
		recorder := factory.NewRecorderPCM()
		if err := recorder.Ping(); err == nil {
			return NewRecorder(recorder)
		}
	}

	for _, factory := range registry.RecorderFactories() {
		recorder := factory.NewRecorderPCM()
		err := recorder.Ping()
		logger.Debugf(ctx, "pinging PCM recorder %T result is %v", recorder, err)
		if err == nil {
			lastSuccessfulRecorderFactoryLocker.Lock()
			defer lastSuccessfulRecorderFactoryLocker.Unlock()
			lastSuccessfulRecorderFactory = factory
			return NewRecorder(recorder)
		}
	}

	logger.Infof(ctx, "was unable to initialize any PCM recorder")
	return &Recorder{
		RecorderPCM: RecorderPCMDummy{},
	}
}

func (a *Recorder) RecordPCM(
	sampleRate SampleRate,
	channels Channel,
	pcmFormat PCMFormat,
	pcmWriter io.Writer,
) (RecordStream, error) {
	return a.RecorderPCM.RecordPCM(
		sampleRate,
		channels,
		pcmFormat,
		pcmWriter,
	)
}
