package pulseaudio

import (
	"fmt"

	"github.com/jfreymuth/pulse"
)

type RecordStream struct {
	*pulse.Client
	*pulse.RecordStream
}

func newRecordStream(
	client *pulse.Client,
	pulseStream *pulse.RecordStream,
) *RecordStream {
	return &RecordStream{
		Client:       client,
		RecordStream: pulseStream,
	}
}

func (stream *RecordStream) Drain() error {
	if stream.Error() != nil {
		return fmt.Errorf("an error occurred during playback: %w", stream.Error())
	}
	return nil
}

func (stream *RecordStream) Close() (err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("got a panic: %v", r)
		}
	}()
	stream.RecordStream.Stop()
	stream.RecordStream.Close()
	stream.Client.Close()
	return
}
