package main

import (
	"context"
	_ "embed"
	"os"
	"time"

	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	"github.com/spf13/pflag"
	"github.com/xaionaro-go/audio/pkg/audio"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/oto"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/portaudio"
	"github.com/xaionaro-go/audio/pkg/audio/backends/pulseaudio"
	"github.com/xaionaro-go/datacounter"
	"github.com/xaionaro-go/observability"
)

func main() {
	loggerLevel := logger.LevelDebug
	pflag.Var(&loggerLevel, "log-level", "Log level")
	pflag.Parse()

	l := logrus.Default().WithLevel(loggerLevel)
	ctx := logger.CtxWithLogger(context.Background(), l)
	logger.Default = func() logger.Logger {
		return l
	}
	defer belt.Flush(ctx)

	logger.Infof(ctx, "starting...")
	recorder := audio.NewRecorderAuto(ctx)
	defer recorder.Close()
	wc := datacounter.NewWriterCounter(os.Stdout)
	logger.Tracef(ctx, "recorder.RecordPCM")
	streamRecord, err := recorder.RecordPCM(ctx, 48000, 2, audio.PCMFormatFloat32LE, wc)
	logger.Tracef(ctx, "/recorder.RecordPCM: %v", err)
	assertNoError(err)
	defer func() {
		assertNoError(streamRecord.Close())
	}()
	observability.Go(ctx, func(ctx context.Context) {
		logger.Tracef(ctx, "started the traffic count printer loop")
		t := time.NewTicker(time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				logger.Debugf(ctx, "written: %d", wc.Count())
				if pulseStreamRecord, ok := streamRecord.(*pulseaudio.RecordStream); ok {
					logger.Debugf(ctx, "record stream status: running:%v, closed:%v, err:%v", pulseStreamRecord.Running(), pulseStreamRecord.Closed(), pulseStreamRecord.Error())
				}
			}
		}
	})
	<-context.Background().Done()
}

func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}
