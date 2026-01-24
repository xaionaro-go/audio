package main

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"net/http"
	_ "net/http/pprof"
	"os"
	"unsafe"

	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	"github.com/spf13/pflag"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/oto"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/portaudio"
	"github.com/xaionaro-go/audio/pkg/noisesuppression/implementations/rnnoise"
	"github.com/xaionaro-go/observability"
)

func main() {
	loggerLevel := logger.LevelDebug
	pflag.Var(&loggerLevel, "log-level", "Log level")
	isS16Flag := pflag.Bool("s16", false, "")
	netPprofAddr := pflag.String("net-pprof-listen-addr", "", "an address to listen for incoming net/pprof connections")
	pflag.Parse()

	if pflag.NArg() != 2 {
		panic(fmt.Errorf("expected exactly two arguments: <input-file> <output-file>"))
	}

	input, err := os.ReadFile(pflag.Arg(0))
	assertNoError(err)

	if *isS16Flag {
		unconvertedInput := unsafe.Slice((*int16)(unsafe.Pointer(&input[0])), len(input)/2)
		convertedInput := make([]float32, len(input)/2)
		for idx := range convertedInput {
			convertedInput[idx] = float32(unconvertedInput[idx]) / math.MaxInt16
		}
		input = unsafe.Slice((*byte)(unsafe.Pointer(&convertedInput[0])), len(input)*2)
	}

	l := logrus.Default().WithLevel(loggerLevel)
	ctx := logger.CtxWithLogger(context.Background(), l)
	logger.Default = func() logger.Logger {
		return l
	}
	defer belt.Flush(ctx)

	if *netPprofAddr != "" {
		observability.Go(ctx, func(ctx context.Context) { l.Error(http.ListenAndServe(*netPprofAddr, nil)) })
	}

	noiseSuppress, err := rnnoise.New(1)
	assertNoError(err)
	defer noiseSuppress.Close()

	chunkSize := noiseSuppress.ChunkSize()

	tailSize := len(input) % int(chunkSize)
	if tailSize != 0 {
		input = append(input, make([]byte, int(chunkSize)-tailSize)...)
	}
	output := make([]byte, len(input))

	_, err = noiseSuppress.SuppressNoise(ctx, input, output)
	assertNoError(err)

	if *isS16Flag {
		unconvertedOutput := unsafe.Slice((*float32)(unsafe.Pointer(&output[0])), len(output)/4)
		convertedOutput := make([]int16, len(output)/4)
		for idx := range convertedOutput {
			convertedOutput[idx] = int16(unconvertedOutput[idx] * math.MaxInt16)
		}
		output = unsafe.Slice((*byte)(unsafe.Pointer(&convertedOutput[0])), len(output)/2)
	}

	err = os.WriteFile(pflag.Arg(1), output, 0640)
	assertNoError(err)
}

func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}
