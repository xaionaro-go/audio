//go:build !rnnoise
// +build !rnnoise

package rnnoise

import (
	"fmt"

	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/noisesuppression"
)

type RNNoise = noisesuppression.Dummy

func New(
	channels audio.Channel,
) (*RNNoise, error) {
	return nil, fmt.Errorf("built without tag 'rnnoise'")
}
