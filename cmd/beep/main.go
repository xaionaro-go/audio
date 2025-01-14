package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"

	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	"github.com/spf13/pflag"
	"github.com/xaionaro-go/audio/pkg/audio"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/oto"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/portaudio"
)

//go:embed resources/long_audio.ogg
var longVorbis []byte

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

	p := audio.NewPlayerAuto(ctx)
	defer p.Close()
	fmt.Printf("using backend %T\n", p.PlayerPCM)
	stream, err := p.PlayVorbis(ctx, bytes.NewReader(longVorbis))
	assertNoError(err)
	assertNoError(stream.Drain())
	assertNoError(stream.Close())
}

func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}
