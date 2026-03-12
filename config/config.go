package config

import (
	"encoding/json"
	"os"
)

// Config 配置结构体
type Config struct {
	PlayCount                  int     `json:"play_count"`
	AntiLockInterval           int     `json:"anti_lock_interval"`
	BluetoothHeartbeatInterval int     `json:"bluetooth_heartbeat_interval"`
	PlayInterval               int     `json:"play_interval"`
	AudioDirectory             string  `json:"audio_directory"`
	HTTPPort                   int     `json:"http_port"`
	Debug                      bool    `json:"debug"`
	// 检测阈值配置
	VolumeThreshold            float64 `json:"volume_threshold"`
	LowFreqRatioThreshold      float64 `json:"low_freq_ratio_threshold"`
	TotalEnergyThreshold       float64 `json:"total_energy_threshold"`
}

// LoadConfig 从默认文件加载配置
func LoadConfig() (Config, error) {
	return LoadConfigFromFile("config.json")
}

// LoadConfigFromFile 从指定文件加载配置
func LoadConfigFromFile(path string) (Config, error) {
	var config Config

	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	return config, err
}
