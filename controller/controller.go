package controller

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"randomVoiceAttack/detector"
	"randomVoiceAttack/logger"
	"randomVoiceAttack/player"
)

type PlaybackController interface {
	IsPlaying() bool
	SetPlaying(playing bool)
}

type AudioController struct {
	AudioFiles []string
	playCount  int
	isPlaying  bool
	playMutex  sync.Mutex
}

func NewAudioController(audioFiles []string, playCount int) *AudioController {
	return &AudioController{
		AudioFiles: audioFiles,
		playCount:  playCount,
		isPlaying:  false,
		playMutex:  sync.Mutex{},
	}
}

func (ac *AudioController) IsPlaying() bool {
	ac.playMutex.Lock()
	defer ac.playMutex.Unlock()
	return ac.isPlaying
}

func (ac *AudioController) SetPlaying(playing bool) {
	ac.playMutex.Lock()
	defer ac.playMutex.Unlock()
	ac.isPlaying = playing
}

func (ac *AudioController) PlayRandomAudios(ctx context.Context) {
	ac.SetPlaying(true)
	defer ac.SetPlaying(false)

	for i := 0; i < ac.playCount; i++ {
		select {
		case <-ctx.Done():
			logger.Info("Playback cancelled")
			return
		default:
		}

		randomFile := ac.AudioFiles[rand.Intn(len(ac.AudioFiles))]
		logger.Info("Playing (%d/%d): %s", i+1, ac.playCount, randomFile)

		err := player.PlayAudioWithContext(ctx, randomFile)
		if err != nil {
			if err == context.Canceled {
				logger.Info("Playback cancelled")
				return
			}
			logger.Info("Error playing audio: %v", err)
			continue
		}
	}
}

func (ac *AudioController) DetectAndPlay(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done():
		return false, nil
	default:
	}

	if ac.IsPlaying() {
		time.Sleep(100 * time.Millisecond)
		return false, nil
	}

	hasLowFreq, err := detector.DetectLowFrequencySound()
	if err != nil {
		logger.Info("Error detecting sound: %v", err)
		time.Sleep(100 * time.Millisecond)
		return false, err
	}

	if hasLowFreq {
		logger.Info("Low frequency noise detected! Playing audio...")
		ac.PlayRandomAudios(ctx)
		logger.Info("Audio playback completed. Listening for low frequency noise...")
	}

	select {
	case <-ctx.Done():
		return hasLowFreq, nil
	default:
	}

	time.Sleep(1000 * time.Millisecond)
	return hasLowFreq, nil
}
