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

// 全局变量，用于跟踪扬声器是否已经初始化
var speakerInitialized bool

// 播放指定的音频文件
func PlayAudio(filePath string) error {
	return PlayAudioWithContext(context.Background(), filePath)
}

// 播放指定的音频文件，支持context取消
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

// 播放静音音频，用于蓝牙心跳
func PlaySilentAudio() error {
	// 初始化扬声器（使用默认采样率）
	sampleRate := beep.SampleRate(44100)
	// 只初始化一次扬声器
	if !speakerInitialized {
		speaker.Init(sampleRate, sampleRate.N(time.Second/10))
		speakerInitialized = true
	}

	// 创建一个简短的静音音频流
	silentStreamer := beep.Silence(sampleRate.N(100 * time.Millisecond))

	// 播放静音音频
	done := make(chan bool)
	speaker.Play(beep.Seq(silentStreamer, beep.Callback(func() {
		done <- true
	})))

	<-done
	return nil
}
