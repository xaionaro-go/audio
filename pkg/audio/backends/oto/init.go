package oto

import (
	"github.com/xaionaro-go/audio/pkg/audio/registry"
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

const (
	Priority = 50
)

func init() {
	registry.RegisterPlayerFactory(Priority, PlayerPCMFactory{})
}

type PlayerPCMFactory struct{}

func (PlayerPCMFactory) NewPlayerPCM() (types.PlayerPCM, error) {
	return NewPlayerPCM()
}
