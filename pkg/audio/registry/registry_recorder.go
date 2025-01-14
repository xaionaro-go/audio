package registry

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/xaionaro-go/audio/pkg/audio/types"
)

type RecorderPCMFactory interface {
	NewRecorderPCM() (types.RecorderPCM, error)
}

type recorderFactoryWithPriority struct {
	Priority int
	RecorderPCMFactory
}

var recorderFactoryRegistry = map[reflect.Type]recorderFactoryWithPriority{}

func RegisterRecorderFactory(
	priority int,
	recorderPCMFactory RecorderPCMFactory,
) {
	t := reflect.ValueOf(recorderPCMFactory).Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if _, ok := recorderFactoryRegistry[t]; ok {
		panic(fmt.Errorf("there is already registered a factory of RecorderPCM of type %v", t))
	}
	recorderFactoryRegistry[t] = recorderFactoryWithPriority{
		Priority:           priority,
		RecorderPCMFactory: recorderPCMFactory,
	}
}

func RecorderFactories() []RecorderPCMFactory {
	var factoriesWithPriorities []recorderFactoryWithPriority
	for _, factory := range recorderFactoryRegistry {
		factoriesWithPriorities = append(factoriesWithPriorities, factory)
	}
	sort.Slice(factoriesWithPriorities, func(i, j int) bool {
		return factoriesWithPriorities[i].Priority > factoriesWithPriorities[j].Priority
	})

	var factories []RecorderPCMFactory
	for _, factory := range factoriesWithPriorities {
		factories = append(factories, factory.RecorderPCMFactory)
	}

	return factories
}
