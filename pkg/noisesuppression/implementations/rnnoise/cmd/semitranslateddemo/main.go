package main

/*
#cgo pkg-config: rnnoise
#cgo CFLAGS: -march=native
#include <rnnoise.h>
*/
import "C"
import (
	"errors"
	"flag"
	"io"
	"os"
	"unsafe"
)

const (
	FRAME_SIZE = 480
)

func main() {
	flag.Parse()

	inputFile, err := os.Open(flag.Arg(0))
	assertNoError(err)
	defer inputFile.Close()

	outputFile, err := os.OpenFile(flag.Arg(1), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0640)
	assertNoError(err)
	defer outputFile.Close()

	var x [FRAME_SIZE]float32
	st := C.rnnoise_create(nil)
	first := true
	for {
		var tmp [FRAME_SIZE]int16
		_, err := inputFile.Read(unsafe.Slice((*byte)(unsafe.Pointer(&tmp[0])), FRAME_SIZE*2))
		if errors.Is(err, io.EOF) {
			break
		}

		for i := 0; i < FRAME_SIZE; i++ {
			x[i] = float32(tmp[i])
		}
		C.rnnoise_process_frame(st, (*C.float)(&x[0]), (*C.float)(&x[0]))
		for i := 0; i < FRAME_SIZE; i++ {
			tmp[i] = int16(x[i])
		}
		if !first {
			outputFile.Write(unsafe.Slice((*byte)(unsafe.Pointer(&tmp[0])), FRAME_SIZE*2))
		}
		first = false
	}
	C.rnnoise_destroy(st)
}

func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}
