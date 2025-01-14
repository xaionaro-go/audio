package audio

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/hashicorp/go-multierror"
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
		player, err := factory.NewPlayerPCM()
		if err == nil {
			if err := player.Ping(ctx); err == nil {
				return NewPlayer(player)
			}
		}
	}

	var mErr *multierror.Error
	for _, factory := range registry.PlayerFactories() {
		player, err := factory.NewPlayerPCM()
		logger.Debugf(ctx, "initializing player %T result is %v", player, err)
		if err != nil {
			mErr = multierror.Append(mErr, fmt.Errorf("unable to initialize %T: %w", player, err))
			continue
		}

		err = player.Ping(ctx)
		logger.Debugf(ctx, "pinging PCM player %T result is %v", player, err)
		if err != nil {
			mErr = multierror.Append(mErr, fmt.Errorf("unable to ping %T: %w", player, err))
			continue
		}

		lastSuccessfulPlayerFactoryLocker.Lock()
		defer lastSuccessfulPlayerFactoryLocker.Unlock()
		lastSuccessfulPlayerFactory = factory
		return NewPlayer(player)
	}

	logger.Infof(ctx, "was unable to initialize any PCM player: %v", mErr.ErrorOrNil())
	return &Player{
		PlayerPCM: PlayerPCMDummy{},
	}
}

func (a *Player) PlayVorbis(
	ctx context.Context,
	rawReader io.Reader,
) (PlayStream, error) {
	oggReader, err := oggvorbis.NewReader(rawReader)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize a vorbis reader: %w", err)
	}

	stream, err := a.PlayerPCM.PlayPCM(
		ctx,
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
	ctx context.Context,
	sampleRate SampleRate,
	channels Channel,
	pcmFormat PCMFormat,
	bufferSize time.Duration,
	pcmReader io.Reader,
) (PlayStream, error) {
	return a.PlayerPCM.PlayPCM(
		ctx,
		sampleRate,
		channels,
		pcmFormat,
		bufferSize,
		pcmReader,
	)
}
