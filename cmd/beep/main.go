package main

import (
	"bytes"
	"context"
	_ "embed"

	"github.com/xaionaro-go/audio/pkg/audio"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/oto"
)

//go:embed resources/long_audio.ogg
var longVorbis []byte

func main() {
	ctx := context.Background()
	a := audio.NewAudioAuto(ctx)
	stream, err := a.PlayVorbis(bytes.NewReader(longVorbis))
	assertNoError(err)
	assertNoError(stream.Drain())
	assertNoError(stream.Close())
}

func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}
