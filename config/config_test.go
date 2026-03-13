// Package config provides configuration management functionality.
package config

import (
	"encoding/json"
	"os"
	"testing"
)

// TestValidate tests the Config.Validate method.
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				HTTPPort:                   8080,
				PlayCount:                  3,
				AudioDirectory:             "./res",
				AntiLockInterval:           100,
				BluetoothHeartbeatInterval: 5000,
				VolumeThreshold:            0.005,
				LowFreqRatioThreshold:      0.015,
				TotalEnergyThreshold:       0.01,
			},
			wantErr: false,
		},
		{
			name: "invalid HTTP port (too low)",
			config: Config{
				HTTPPort:       0,
				PlayCount:      3,
				AudioDirectory: "./res",
			},
			wantErr: true,
		},
		{
			name: "invalid HTTP port (too high)",
			config: Config{
				HTTPPort:       65536,
				PlayCount:      3,
				AudioDirectory: "./res",
			},
			wantErr: true,
		},
		{
			name: "invalid play count (too low)",
			config: Config{
				HTTPPort:       8080,
				PlayCount:      0,
				AudioDirectory: "./res",
			},
			wantErr: true,
		},
		{
			name: "invalid play count (too high)",
			config: Config{
				HTTPPort:       8080,
				PlayCount:      101,
				AudioDirectory: "./res",
			},
			wantErr: true,
		},
		{
			name: "empty audio directory",
			config: Config{
				HTTPPort:       8080,
				PlayCount:      3,
				AudioDirectory: "",
			},
			wantErr: true,
		},
		{
			name: "invalid anti-lock interval",
			config: Config{
				HTTPPort:         8080,
				PlayCount:        3,
				AudioDirectory:   "./res",
				AntiLockInterval: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid Bluetooth heartbeat interval",
			config: Config{
				HTTPPort:                   8080,
				PlayCount:                  3,
				AudioDirectory:             "./res",
				BluetoothHeartbeatInterval: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid volume threshold",
			config: Config{
				HTTPPort:        8080,
				PlayCount:       3,
				AudioDirectory:  "./res",
				VolumeThreshold: -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid low frequency ratio threshold (too low)",
			config: Config{
				HTTPPort:              8080,
				PlayCount:             3,
				AudioDirectory:        "./res",
				LowFreqRatioThreshold: -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid low frequency ratio threshold (too high)",
			config: Config{
				HTTPPort:              8080,
				PlayCount:             3,
				AudioDirectory:        "./res",
				LowFreqRatioThreshold: 1.1,
			},
			wantErr: true,
		},
		{
			name: "invalid total energy threshold",
			config: Config{
				HTTPPort:             8080,
				PlayCount:            3,
				AudioDirectory:       "./res",
				TotalEnergyThreshold: -0.1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLoadConfigFromFile tests loading config from a file.
func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary test config file
	tempFile, err := os.CreateTemp("", "config_test*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	testConfig := Config{
		HTTPPort:                   9999,
		PlayCount:                  5,
		AudioDirectory:             "./test",
		AntiLockInterval:           200,
		BluetoothHeartbeatInterval: 10000,
		Debug:                      true,
		VolumeThreshold:            0.01,
		LowFreqRatioThreshold:      0.02,
		TotalEnergyThreshold:       0.05,
	}

	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(tempFile.Name(), data, 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test loading the config
	loadedConfig, err := LoadConfigFromFile(tempFile.Name())
	if err != nil {
		t.Fatalf("LoadConfigFromFile failed: %v", err)
	}

	// Verify the loaded config
	if loadedConfig.HTTPPort != testConfig.HTTPPort {
		t.Errorf("HTTPPort mismatch: got %d, want %d", loadedConfig.HTTPPort, testConfig.HTTPPort)
	}
	if loadedConfig.PlayCount != testConfig.PlayCount {
		t.Errorf("PlayCount mismatch: got %d, want %d", loadedConfig.PlayCount, testConfig.PlayCount)
	}
	if loadedConfig.AudioDirectory != testConfig.AudioDirectory {
		t.Errorf("AudioDirectory mismatch: got %s, want %s", loadedConfig.AudioDirectory, testConfig.AudioDirectory)
	}
	if loadedConfig.Debug != testConfig.Debug {
		t.Errorf("Debug mismatch: got %v, want %v", loadedConfig.Debug, testConfig.Debug)
	}
}

// TestLoadConfigFromFile_InvalidFile tests loading config from a non-existent file.
func TestLoadConfigFromFile_InvalidFile(t *testing.T) {
	_, err := LoadConfigFromFile("non_existent_file.json")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestLoadConfigFromFile_InvalidJSON tests loading config from invalid JSON.
func TestLoadConfigFromFile_InvalidJSON(t *testing.T) {
	tempFile, err := os.CreateTemp("", "config_test*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write invalid JSON
	if err := os.WriteFile(tempFile.Name(), []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	_, err = LoadConfigFromFile(tempFile.Name())
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}
