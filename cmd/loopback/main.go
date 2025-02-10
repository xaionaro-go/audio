package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	"github.com/spf13/pflag"
	"github.com/xaionaro-go/audio/pkg/audio"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/oto"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/portaudio"
	"github.com/xaionaro-go/audio/pkg/audio/backends/pulseaudio"
	"github.com/xaionaro-go/audio/pkg/noisesuppression/implementations/rnnoise"
	"github.com/xaionaro-go/audio/pkg/noisesuppressionstream"
	"github.com/xaionaro-go/datacounter"
	"github.com/xaionaro-go/observability"
)

func main() {
	loggerLevel := logger.LevelDebug
	pflag.Var(&loggerLevel, "log-level", "Log level")
	netPprofAddr := pflag.String("net-pprof-listen-addr", "", "an address to listen for incoming net/pprof connections")
	noiseSuppressionFlag := pflag.Bool("noise-suppression", false, "enable noise suppression using RNNoise")
	pflag.Parse()

	l := logrus.Default().WithLevel(loggerLevel)
	ctx := logger.CtxWithLogger(context.Background(), l)
	logger.Default = func() logger.Logger {
		return l
	}
	defer belt.Flush(ctx)

	if *netPprofAddr != "" {
		observability.Go(ctx, func() { l.Error(http.ListenAndServe(*netPprofAddr, nil)) })
	}

	logger.Infof(ctx, "starting...")
	recorder := audio.NewRecorderAuto(ctx)
	defer recorder.Close()

	player := audio.NewPlayerAuto(ctx)
	defer player.Close()

	var (
		r io.Reader
		w io.Writer
	)
	r, w = io.Pipe()
	wc := datacounter.NewWriterCounter(w)

	logger.Tracef(ctx, "recorder.RecordPCM")
	streamRecord, err := recorder.RecordPCM(ctx, 48000, 2, audio.PCMFormatFloat32LE, wc)
	logger.Tracef(ctx, "/recorder.RecordPCM: %v", err)
	assertNoError(err)
	defer func() {
		assertNoError(streamRecord.Close())
	}()

	if *noiseSuppressionFlag {
		logger.Tracef(ctx, "rnnoise.New(2)")
		noiseSuppressor, err := rnnoise.New(2)
		logger.Tracef(ctx, "/rnnoise.New(2): %v", err)
		assertNoError(err)

		enc, err := noiseSuppressor.Encoding(ctx)
		assertNoError(err)
		encPCM := enc.(*audio.EncodingPCM)

		if encPCM.PCMFormat != audio.PCMFormatFloat32LE {
			panic(fmt.Errorf("unexpected PCM format on the noise suppression: %v", encPCM.PCMFormat))
		}

		logger.Tracef(ctx, "noisesuppressionstream.NewNoiseSuppressionStream")
		r, err = noisesuppressionstream.NewNoiseSuppressionStream(
			ctx, r, noiseSuppressor, 1024*1024, 1024*1024,
		)
		logger.Tracef(ctx, "/noisesuppressionstream.NewNoiseSuppressionStream: %v", err)
		assertNoError(err)
	}

	observability.Go(ctx, func() {
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

	logger.Tracef(ctx, "player.PlayPCM")
	streamPlay, err := player.PlayPCM(ctx, 48000, 2, audio.PCMFormatFloat32LE, 300*time.Millisecond, r)
	logger.Tracef(ctx, "/player.PlayPCM: %v", err)
	assertNoError(err)
	defer streamPlay.Close()

	logger.Infof(ctx, "started (%T -> %T)", recorder.RecorderPCM, player.PlayerPCM)
	streamPlay.Drain()
	defer func() {
		assertNoError(streamPlay.Close())
	}()
	<-context.Background().Done()
}

func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}
