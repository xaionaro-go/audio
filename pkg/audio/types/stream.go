package types

import (
	"io"
)

type Stream interface {
	io.Closer
}

type PlayStream interface {
	Stream
	Drain() error
}

type RecordStream interface {
	Stream
}
