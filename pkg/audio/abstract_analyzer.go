package audio

import (
	"context"
	"io"
)

type AbstractAnalyzer interface {
	io.Closer

	Encoding(context.Context) (Encoding, error)
	Channels(context.Context) (Channel, error)
}

/* for easier copy&paste:

func () Close() error {
}

func () Encoding(
	ctx context.Context,
) (audio.Encoding, error) {
}

func () Channels(
	ctx context.Context,
) (audio.Channel, error) {
}

*/
