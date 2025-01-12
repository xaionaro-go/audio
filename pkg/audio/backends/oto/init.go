package oto

import (
	"github.com/xaionaro-go/audio/pkg/audio/registry"
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

const (
	Priority = 50
)

func init() {
	registry.RegisterPlayerFactory(Priority, PlayerPCMOtoFactory{})
}

type PlayerPCMOtoFactory struct{}

func (PlayerPCMOtoFactory) NewPlayerPCM() types.PlayerPCM {
	return NewPlayerPCM()
}
