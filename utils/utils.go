package utils

import (
	"io/ioutil"
	"path/filepath"
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
