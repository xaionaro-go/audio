package main

/*
#cgo pkg-config: rnnoise
#cgo CFLAGS: -march=native
#include <rnnoise.h>
*/
import "C"
import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"unsafe"

	"github.com/xaionaro-go/audio/pkg/noisesuppression/implementations/rnnoise"
)

const (
	FRAME_SIZE = 480
)

func main() {
	flag.Parse()
	ctx := context.Background()

	inputFile, err := os.Open(flag.Arg(0))
	assertNoError(err)
	defer inputFile.Close()

	outputFile, err := os.OpenFile(flag.Arg(1), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0640)
	assertNoError(err)
	defer outputFile.Close()

	ns, err := rnnoise.New(1)
	assertNoError(err)
	defer ns.Close()

	first := true
	var x [FRAME_SIZE]float32
	for {
		var tmp [FRAME_SIZE]int16
		_, err := inputFile.Read(unsafe.Slice((*byte)(unsafe.Pointer(&tmp[0])), FRAME_SIZE*2))
		if errors.Is(err, io.EOF) {
			break
		}

		for i := 0; i < FRAME_SIZE; i++ {
			x[i] = float32(tmp[i])
		}
		frame := unsafe.Slice((*byte)(unsafe.Pointer(&x[0])), FRAME_SIZE*4)
		ns.SuppressNoise(ctx, frame, frame)
		for i := 0; i < FRAME_SIZE; i++ {
			tmp[i] = int16(x[i])
		}
		if !first {
			outputFile.Write(unsafe.Slice((*byte)(unsafe.Pointer(&tmp[0])), FRAME_SIZE*2))
		}
		first = false
	}
}

func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}
