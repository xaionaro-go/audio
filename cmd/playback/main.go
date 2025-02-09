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
)

func main() {
	loggerLevel := logger.LevelDebug
	pflag.Var(&loggerLevel, "log-level", "Log level")
	pflag.Parse()

	if pflag.NArg() != 1 {
		panic("expected exactly one positional argument: path to the float32le 48K 2ch WAV file")
	}
	filePath := pflag.Arg(0)

	l := logrus.Default().WithLevel(loggerLevel)
	ctx := logger.CtxWithLogger(context.Background(), l)
	logger.Default = func() logger.Logger {
		return l
	}
	defer belt.Flush(ctx)

	logger.Infof(ctx, "starting...")
	file, err := os.Open(filePath)
	assertNoError(err)
	defer file.Close()

	player := audio.NewPlayerAuto(ctx)
	defer player.Close()
	logger.Tracef(ctx, "player.PlayPCM")
	streamPlay, err := player.PlayPCM(ctx, 48000, 2, audio.PCMFormatFloat32LE, 100*time.Millisecond, file)
	logger.Tracef(ctx, "/player.PlayPCM: %v", err)
	assertNoError(err)
	defer streamPlay.Close()
	logger.Infof(ctx, "started (file -> %T)", player.PlayerPCM)
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
