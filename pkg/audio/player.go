package audio

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/jfreymuth/oggvorbis"
	"github.com/xaionaro-go/audio/pkg/audio/registry"
)

const BufferSize = 100 * time.Millisecond

type Player struct {
	PlayerPCM
}

func NewPlayer(playerPCM PlayerPCM) *Player {
	return &Player{
		PlayerPCM: playerPCM,
	}
}

var (
	lastSuccessfulPlayerFactory       registry.PlayerPCMFactory
	lastSuccessfulPlayerFactoryLocker sync.Mutex
)

func getLastSuccessfulPlayerFactory() registry.PlayerPCMFactory {
	lastSuccessfulPlayerFactoryLocker.Lock()
	defer lastSuccessfulPlayerFactoryLocker.Unlock()
	return lastSuccessfulPlayerFactory
}

func NewPlayerAuto(
	ctx context.Context,
) *Player {
	factory := getLastSuccessfulPlayerFactory()
	if factory != nil {
		player := factory.NewPlayerPCM()
		if err := player.Ping(); err == nil {
			return NewPlayer(player)
		}
	}

	for _, factory := range registry.PlayerFactories() {
		player := factory.NewPlayerPCM()
		err := player.Ping()
		logger.Debugf(ctx, "pinging PCM player %T result is %v", player, err)
		if err == nil {
			lastSuccessfulPlayerFactoryLocker.Lock()
			defer lastSuccessfulPlayerFactoryLocker.Unlock()
			lastSuccessfulPlayerFactory = factory
			return NewPlayer(player)
		}
	}

	logger.Infof(ctx, "was unable to initialize any PCM player")
	return &Player{
		PlayerPCM: PlayerPCMDummy{},
	}
}

func (a *Player) PlayVorbis(rawReader io.Reader) (PlayStream, error) {
	oggReader, err := oggvorbis.NewReader(rawReader)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize a vorbis reader: %w", err)
	}

	stream, err := a.PlayerPCM.PlayPCM(
		SampleRate(oggReader.SampleRate()),
		Channel(oggReader.Channels()),
		PCMFormatFloat32LE,
		BufferSize,
		newReaderFromFloat32Reader(oggReader),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to playback as PCM: %w", err)
	}
	return stream, nil
}

func (a *Player) PlayPCM(
	sampleRate SampleRate,
	channels Channel,
	pcmFormat PCMFormat,
	bufferSize time.Duration,
	pcmReader io.Reader,
) (PlayStream, error) {
	return a.PlayerPCM.PlayPCM(
		sampleRate,
		channels,
		pcmFormat,
		bufferSize,
		pcmReader,
	)
}
