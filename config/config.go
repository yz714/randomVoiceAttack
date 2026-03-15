package config

import (
	"encoding/json"
	"errors"
	"os"
)

const (
	minHTTPPort = 1
	maxHTTPPort = 65535
	minPlayCount = 1
	maxPlayCount = 100
	minIntervalMs = 1
)

var (
	ErrInvalidHTTPPort = errors.New("HTTP port must be between 1 and 65535")
	ErrInvalidPlayCount = errors.New("play count must be between 1 and 100")
	ErrEmptyAudioDirectory = errors.New("audio directory cannot be empty")
	ErrInvalidAntiLockInterval = errors.New("anti-lock interval must be at least 1ms")
	ErrInvalidBluetoothHeartbeatInterval = errors.New("bluetooth heartbeat interval must be at least 1ms")
	ErrInvalidVolumeThreshold = errors.New("volume threshold must be non-negative")
	ErrInvalidLowFreqRatioThreshold = errors.New("low frequency ratio threshold must be between 0 and 1")
	ErrInvalidTotalEnergyThreshold = errors.New("total energy threshold must be non-negative")
	ErrInvalidNoiseDataRetentionDays = errors.New("noise data retention days must be at least 1")
)

type Config struct {
	PlayCount                  int     `json:"play_count"`
	AntiLockInterval           int     `json:"anti_lock_interval"`
	BluetoothHeartbeatInterval int     `json:"bluetooth_heartbeat_interval"`
	PlayInterval               int     `json:"play_interval"`
	AudioDirectory             string  `json:"audio_directory"`
	HTTPPort                   int     `json:"http_port"`
	Debug                      bool    `json:"debug"`
	VolumeThreshold            float64 `json:"volume_threshold"`
	LowFreqRatioThreshold      float64 `json:"low_freq_ratio_threshold"`
	TotalEnergyThreshold       float64 `json:"total_energy_threshold"`
	NoiseDataRetentionDays     int     `json:"noise_data_retention_days"`
}

func LoadConfig() (Config, error) {
	return LoadConfigFromFile("config.json")
}

func LoadConfigFromFile(path string) (Config, error) {
	var config Config

	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}

	if err := config.Validate(); err != nil {
		return config, err
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.HTTPPort < minHTTPPort || c.HTTPPort > maxHTTPPort {
		return ErrInvalidHTTPPort
	}

	if c.PlayCount < minPlayCount || c.PlayCount > maxPlayCount {
		return ErrInvalidPlayCount
	}

	if c.AudioDirectory == "" {
		return ErrEmptyAudioDirectory
	}

	if c.AntiLockInterval < minIntervalMs {
		return ErrInvalidAntiLockInterval
	}

	if c.BluetoothHeartbeatInterval < minIntervalMs {
		return ErrInvalidBluetoothHeartbeatInterval
	}

	if c.VolumeThreshold < 0 {
		return ErrInvalidVolumeThreshold
	}

	if c.LowFreqRatioThreshold < 0 || c.LowFreqRatioThreshold > 1 {
		return ErrInvalidLowFreqRatioThreshold
	}

	if c.TotalEnergyThreshold < 0 {
		return ErrInvalidTotalEnergyThreshold
	}

	if c.NoiseDataRetentionDays < 1 {
		return ErrInvalidNoiseDataRetentionDays
	}

	return nil
}

func (c *Config) GetMaxNoiseDataEntries() int {
	const entriesPerSecond = 1
	const secondsPerMinute = 60
	const minutesPerHour = 60
	const hoursPerDay = 24
	return c.NoiseDataRetentionDays * hoursPerDay * minutesPerHour * secondsPerMinute * entriesPerSecond
}
