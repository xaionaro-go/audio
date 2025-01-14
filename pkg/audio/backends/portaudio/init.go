package portaudio

import (
	"github.com/xaionaro-go/audio/pkg/audio/registry"
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

const (
	Priority = 60
)

func init() {
	registry.RegisterPlayerFactory(Priority, PlayerPCMFactory{})
	registry.RegisterRecorderFactory(Priority, RecorderPCMFactory{})
}

type PlayerPCMFactory struct{}

func (PlayerPCMFactory) NewPlayerPCM() types.PlayerPCM {
	return NewPlayerPCM()
}

type RecorderPCMFactory struct{}

func (RecorderPCMFactory) NewRecorderPCM() types.RecorderPCM {
	return NewRecorderPCM()
}
