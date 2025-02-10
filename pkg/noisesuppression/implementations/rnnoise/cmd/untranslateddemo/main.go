package main

/*
#cgo pkg-config: rnnoise
#cgo CFLAGS: -march=native
#include <stdlib.h>
#include <stdio.h>
#include <rnnoise.h>

#define FRAME_SIZE 480

int run(const char *input_file, const char *output_file) {
  int i;
  int first = 1;
  float x[FRAME_SIZE];
  FILE *f1, *fout;
  DenoiseState *st;
#ifdef USE_WEIGHTS_FILE
  RNNModel *model = rnnoise_model_from_filename("weights_blob.bin");
  st = rnnoise_create(model);
#else
  st = rnnoise_create(NULL);
#endif

  f1 = fopen(input_file, "rb");
  fout = fopen(output_file, "wb");
  while (1) {
    short tmp[FRAME_SIZE];
    fread(tmp, sizeof(short), FRAME_SIZE, f1);
    if (feof(f1)) break;
    for (i=0;i<FRAME_SIZE;i++) x[i] = tmp[i];
    rnnoise_process_frame(st, x, x);
    for (i=0;i<FRAME_SIZE;i++) tmp[i] = x[i];
    if (!first) fwrite(tmp, sizeof(short), FRAME_SIZE, fout);
    first = 0;
  }
  rnnoise_destroy(st);
  fclose(f1);
  fclose(fout);
#ifdef USE_WEIGHTS_FILE
  rnnoise_model_free(model);
#endif
  return 0;
}
*/
import "C"
import (
	"flag"
	"runtime"
	"unsafe"
)

func main() {
	flag.Parse()
	inputFile := C.CString(flag.Arg(0))
	defer C.free(unsafe.Pointer(inputFile))
	outputFile := C.CString(flag.Arg(1))
	defer C.free(unsafe.Pointer(outputFile))
	C.run(inputFile, outputFile)
	runtime.KeepAlive(inputFile)
	runtime.KeepAlive(outputFile)
}
