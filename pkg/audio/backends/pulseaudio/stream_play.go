package pulseaudio

import (
	"fmt"

	"github.com/jfreymuth/pulse"
)

type PlayStream struct {
	*pulse.Client
	*pulse.PlaybackStream
}

func newPlayStream(
	client *pulse.Client,
	pulseStream *pulse.PlaybackStream,
) *PlayStream {
	return &PlayStream{
		Client:         client,
		PlaybackStream: pulseStream,
	}
}

func (stream *PlayStream) Drain() error {
	stream.PlaybackStream.Drain()
	if stream.Error() != nil {
		return fmt.Errorf("an error occurred during playback: %w", stream.Error())
	}
	if stream.Underflow() {
		return fmt.Errorf("underflow")
	}
	return nil
}

func (stream *PlayStream) Close() (err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("got a panic: %v", r)
		}
	}()
	stream.PlaybackStream.Stop()
	stream.PlaybackStream.Close()
	stream.Client.Close()
	return
}
