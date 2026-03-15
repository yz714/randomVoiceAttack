package utils

import (
	"io/ioutil"
	"path/filepath"

	"randomVoiceAttack/logger"
)

// 获取指定目录下的所有音频文件
func GetAudioFiles(dir string) ([]string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var audioFiles []string
	for _, file := range files {
		if !file.IsDir() {
			ext := filepath.Ext(file.Name())
			if ext == ".mp3" {
				audioFiles = append(audioFiles, filepath.Join(dir, file.Name()))
			}
			if ext == ".wav" {
				audioFiles = append(audioFiles, filepath.Join(dir, file.Name()))
			}
		}
	}

	return audioFiles, nil
}

// Go 启动一个带 recover 保护的 goroutine
func Go(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Warn("Recovered from panic in goroutine: %v", r)
			}
		}()
		fn()
	}()
}

// GoWithName 启动一个带 recover 保护和名称的 goroutine，方便调试
func GoWithName(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Warn("Recovered from panic in goroutine [%s]: %v", name, r)
			}
		}()
		fn()
	}()
}
