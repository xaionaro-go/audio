package pulseaudio

import (
	"github.com/xaionaro-go/audio/pkg/audio/registry"
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

const (
	Priority = 100
)

func init() {
	registry.RegisterPlayerFactory(Priority, PlayerPCMFactory{})
	registry.RegisterRecorderFactory(Priority, RecorderPCMFactory{})
}

type PlayerPCMFactory struct{}

func (PlayerPCMFactory) NewPlayerPCM() (types.PlayerPCM, error) {
	return NewPlayerPCM()
}

type RecorderPCMFactory struct{}

func (RecorderPCMFactory) NewRecorderPCM() (types.RecorderPCM, error) {
	return NewRecorderPCM()
}
