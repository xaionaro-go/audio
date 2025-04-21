# `audio`

`audio` is a collection of package to handle audio inputs, outputs and processing in Go.

It currently supports 3 backends:
* [`oto`](./pkg/audio/backends/oto) (https://github.com/ebitengine/oto) [for all OSes, but only playback]
* [`portaudio`](./pkg/audio/backends/portaudio) (https://github.com/gordonklaus/portaudio) [for Windows]
* [`pulseaudio`](./pkg/audio/backends/pulseaudio) (github.com/jfreymuth/pulse) [for Linux]

And it has various modules for audio processing:
* Basics: [`resampler`](./pkg/audio/resampler), [`planar`](./pkg/audio/planar).
* [Noise suppression](./pkg/noisesuppression), also in [streaming mode](./pkg/noisesuppressionstream).
* [Voice Activity Detector](./pkg/vad)

# Examples

**BEEP** using a vorbis audio file
```go
import (
	...

	"github.com/xaionaro-go/audio/pkg/audio"

	// select the backends you want to use:
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/oto"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/portaudio"
)


func beep(...) {
	...
	player := audio.NewPlayerAuto(ctx)
	defer player.Close()
	stream, err := player.PlayVorbis(ctx, vorbisReader) // or use PlayPCM if the byte stream is PCM
	...
}
```
To use a specific backend:
```go
	pulsePCMPlayer := pulse.NewPlayerPCM()
	player := audio.NewPlayer(pulsePCMPlayer)
	defer player.Close()
	stream, err := player.PlayVorbis(ctx, vorbisReader)
```

**RECORD**
```go
import (
	...

	"github.com/xaionaro-go/audio/pkg/audio"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/portaudio"
	_ "github.com/xaionaro-go/audio/pkg/audio/backends/pulseaudio"
)

func record5Seconds(ctx context.Context, w io.Writer) {
	ctx, cancelFn := context.WithCancel(ctx)
	recorder := audio.NewRecorderAuto(ctx)
	defer recorder.Close()
	streamRecord, err := recorder.RecordPCM(ctx, 48000, 2, audio.PCMFormatFloat32LE, w)
	defer streamRecord.Close()
	time.Sleep(5 * time.Second)
	cancelFn()
}
```
