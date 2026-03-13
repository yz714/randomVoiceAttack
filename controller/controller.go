// Package controller provides audio playback control functionality.
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

// PlaybackController defines the interface for audio playback control.
type PlaybackController interface {
	// IsPlaying returns whether audio is currently playing.
	IsPlaying() bool
	// SetPlaying sets the playing status.
	SetPlaying(playing bool)
	// Stop stops the currently playing audio.
	Stop()
	// PlayRandomAudios plays random audio files.
	PlayRandomAudios(ctx context.Context)
}

// AudioController manages audio playback operations.
type AudioController struct {
	AudioFiles      []string
	playCount       int
	isPlaying       bool
	playMutex       sync.Mutex
	playCtx         context.Context
	playCancel      context.CancelFunc
	cancelMutex     sync.Mutex
	isStopped       bool
	stoppedMutex    sync.Mutex
	inCooldown      bool
	cooldownMutex   sync.Mutex
}

// NewAudioController creates a new AudioController instance.
func NewAudioController(audioFiles []string, playCount int) *AudioController {
	return &AudioController{
		AudioFiles:    audioFiles,
		playCount:     playCount,
		isPlaying:     false,
		playMutex:     sync.Mutex{},
		isStopped:     false,
		stoppedMutex:  sync.Mutex{},
		inCooldown:    false,
		cooldownMutex: sync.Mutex{},
	}
}

// IsStopped returns whether the controller is in stopped state.
func (ac *AudioController) IsStopped() bool {
	ac.stoppedMutex.Lock()
	defer ac.stoppedMutex.Unlock()
	return ac.isStopped
}

// SetStopped sets the stopped state.
func (ac *AudioController) SetStopped(stopped bool) {
	ac.stoppedMutex.Lock()
	defer ac.stoppedMutex.Unlock()
	ac.isStopped = stopped
}

// IsInCooldown returns whether the controller is in cooldown period.
func (ac *AudioController) IsInCooldown() bool {
	ac.cooldownMutex.Lock()
	defer ac.cooldownMutex.Unlock()
	return ac.inCooldown
}

// SetInCooldown sets the cooldown state.
func (ac *AudioController) SetInCooldown(inCooldown bool) {
	ac.cooldownMutex.Lock()
	defer ac.cooldownMutex.Unlock()
	ac.inCooldown = inCooldown
}

// IsPlaying returns whether audio is currently playing.
func (ac *AudioController) IsPlaying() bool {
	ac.playMutex.Lock()
	defer ac.playMutex.Unlock()
	return ac.isPlaying
}

// SetPlaying sets the playing status.
func (ac *AudioController) SetPlaying(playing bool) {
	ac.playMutex.Lock()
	defer ac.playMutex.Unlock()
	ac.isPlaying = playing
}

// Stop stops the currently playing audio by canceling the context.
func (ac *AudioController) Stop() {
	ac.SetStopped(true)
	ac.SetInCooldown(false)
	ac.cancelMutex.Lock()
	if ac.playCancel != nil {
		ac.playCancel()
	}
	ac.cancelMutex.Unlock()
}

// PlayRandomAudios plays random audio files from the audio files list.
// It will play the specified number of audio files.
func (ac *AudioController) PlayRandomAudios(ctx context.Context) {
	ac.SetPlaying(true)
	defer ac.SetPlaying(false)

	ac.cancelMutex.Lock()
	ac.playCtx, ac.playCancel = context.WithCancel(ctx)
	localCtx := ac.playCtx
	localCancel := ac.playCancel
	ac.cancelMutex.Unlock()
	defer func() {
		ac.cancelMutex.Lock()
		if localCancel != nil {
			localCancel()
		}
		ac.playCancel = nil
		ac.playCtx = nil
		ac.cancelMutex.Unlock()
	}()

	for i := 0; i < ac.playCount; i++ {
		select {
		case <-localCtx.Done():
			logger.Info("Playback cancelled")
			return
		default:
		}

		randomFile := ac.AudioFiles[rand.Intn(len(ac.AudioFiles))]
		logger.Info("Playing (%d/%d): %s", i+1, ac.playCount, randomFile)

		err := player.PlayAudioWithContext(localCtx, randomFile)
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

// DetectAndPlay detects low frequency sound and plays audio if detected.
// It returns whether low frequency sound was detected and any error encountered.
func (ac *AudioController) DetectAndPlay(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done():
		return false, nil
	default:
	}

	if ac.IsStopped() {
		ac.SetStopped(false)
		time.Sleep(100 * time.Millisecond)
		return false, nil
	}

	if ac.IsInCooldown() {
		time.Sleep(100 * time.Millisecond)
		return false, nil
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
		
		select {
		case <-ctx.Done():
			return hasLowFreq, nil
		default:
		}
		
		logger.Info("Audio playback completed. Entering cooldown period...")
		ac.SetInCooldown(true)

		go func() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
				ac.SetInCooldown(false)
				logger.Info("Cooldown period ended. Listening for low frequency noise...")
			}
		}()
	}

	select {
	case <-ctx.Done():
		return hasLowFreq, nil
	case <-time.After(1000 * time.Millisecond):
	}

	return hasLowFreq, nil
}
