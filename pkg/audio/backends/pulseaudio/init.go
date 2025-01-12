package pulseaudio

import (
	"github.com/xaionaro-go/audio/pkg/audio/registry"
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

const (
	Priority = 100
)

func init() {
	registry.RegisterPlayerFactory(Priority, PlayerPCMPulseFactory{})
	registry.RegisterRecorderFactory(Priority, RecorderPCMPulseFactory{})
}

type PlayerPCMPulseFactory struct{}

func (PlayerPCMPulseFactory) NewPlayerPCM() types.PlayerPCM {
	return NewPlayerPCM()
}

type RecorderPCMPulseFactory struct{}

func (RecorderPCMPulseFactory) NewRecorderPCM() types.RecorderPCM {
	return NewRecorderPCM()
}
