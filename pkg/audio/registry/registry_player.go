package registry

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/xaionaro-go/audio/pkg/audio/types"
)

type PlayerPCMFactory interface {
	NewPlayerPCM() (types.PlayerPCM, error)
}

type playerFactoryWithPriority struct {
	Priority int
	PlayerPCMFactory
}

var playerFactoryRegistry = map[reflect.Type]playerFactoryWithPriority{}

func RegisterPlayerFactory(
	priority int,
	playerPCMFactory PlayerPCMFactory,
) {
	t := reflect.ValueOf(playerPCMFactory).Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if _, ok := playerFactoryRegistry[t]; ok {
		panic(fmt.Errorf("there is already registered a factory of PlayerPCM of type %v", t))
	}
	playerFactoryRegistry[t] = playerFactoryWithPriority{
		Priority:         priority,
		PlayerPCMFactory: playerPCMFactory,
	}
}

func PlayerFactories() []PlayerPCMFactory {
	var factoriesWithPriorities []playerFactoryWithPriority
	for _, factory := range playerFactoryRegistry {
		factoriesWithPriorities = append(factoriesWithPriorities, factory)
	}
	sort.Slice(factoriesWithPriorities, func(i, j int) bool {
		return factoriesWithPriorities[i].Priority > factoriesWithPriorities[j].Priority
	})

	var factories []PlayerPCMFactory
	for _, factory := range factoriesWithPriorities {
		factories = append(factories, factory.PlayerPCMFactory)
	}

	return factories
}
