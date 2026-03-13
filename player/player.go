// Package player provides audio playback functionality using the beep audio library.
package player

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

// Player defines the interface for audio playback operations.
type Player interface {
	// PlayAudio plays an audio file with the default context.
	PlayAudio(filePath string) error
	// PlayAudioWithContext plays an audio file with support for context cancellation.
	PlayAudioWithContext(ctx context.Context, filePath string) error
	// PlaySilentAudio plays a short silent audio for Bluetooth heartbeat.
	PlaySilentAudio() error
}

// speakerInitialized tracks whether the speaker has been initialized.
var speakerInitialized bool

// PlayAudio plays an audio file with the default background context.
func PlayAudio(filePath string) error {
	return PlayAudioWithContext(context.Background(), filePath)
}

// PlayAudioWithContext plays an audio file with support for context cancellation.
// It supports both MP3 and WAV audio formats.
func PlayAudioWithContext(ctx context.Context, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// 根据文件扩展名选择合适的解码器
	ext := filepath.Ext(filePath)
	var streamer beep.StreamSeekCloser
	var format beep.Format

	switch ext {
	case ".mp3":
		streamer, format, err = mp3.Decode(f)
	case ".wav":
		streamer, format, err = wav.Decode(f)
	default:
		return fmt.Errorf("unsupported audio format: %s", ext)
	}

	if err != nil {
		return err
	}
	defer streamer.Close()

	// 只初始化一次扬声器
	if !speakerInitialized {
		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		speakerInitialized = true
	}

	// 等待音频播放完成或取消
	done := make(chan bool, 1)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	select {
	case <-ctx.Done():
		speaker.Clear()
		return ctx.Err()
	case <-done:
		return nil
	}
}

// PlaySilentAudio plays a short silent audio for Bluetooth heartbeat.
// This helps keep the Bluetooth connection alive.
func PlaySilentAudio() error {
	// Initialize speaker with default sample rate
	sampleRate := beep.SampleRate(44100)
	// Initialize speaker only once
	if !speakerInitialized {
		speaker.Init(sampleRate, sampleRate.N(time.Second/10))
		speakerInitialized = true
	}

	// Create a short silent audio stream
	silentStreamer := beep.Silence(sampleRate.N(100 * time.Millisecond))

	// Play the silent audio
	done := make(chan bool)
	speaker.Play(beep.Seq(silentStreamer, beep.Callback(func() {
		done <- true
	})))

	<-done
	return nil
}
