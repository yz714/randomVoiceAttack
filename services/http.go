package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"randomVoiceAttack/logger"
	"randomVoiceAttack/player"
)

// NoiseData 噪音数据结构体
type NoiseData struct {
	Timestamp    string    `json:"timestamp"`
	LowFreqRatio float64   `json:"low_freq_ratio"`
	Volume       float64   `json:"volume"`
	MaxSample    float64   `json:"max_sample"`
	Time         time.Time `json:"time"`
}

// DetectionLog 检测日志结构体
type DetectionLog struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Type      string `json:"type"` // "detection", "playback", "info"
}

// HTTPConfig HTTP服务器配置
type HTTPConfig struct {
	HTTPPort int
}

// HTTPServer HTTP服务器结构体
type HTTPServer struct {
	config          HTTPConfig
	noiseData       []NoiseData
	recentNoiseData []NoiseData
	detectionLogs   []DetectionLog
	audioFiles      []string
	dataMutex       sync.Mutex
	isPlaying       *bool
	playMutex       *sync.Mutex
	dataFile        string
}

// NewHTTPServer 创建新的HTTP服务器
func NewHTTPServer(config HTTPConfig, audioFiles []string, isPlaying *bool, playMutex *sync.Mutex) *HTTPServer {
	server := &HTTPServer{
		config:     config,
		audioFiles: audioFiles,
		isPlaying:  isPlaying,
		playMutex:  playMutex,
		dataFile:   "./noise_data.json",
	}

	// 从文件加载噪音数据
	server.LoadNoiseDataFromFile()

	return server
}

// Start 启动HTTP服务器
func (s *HTTPServer) Start(ctx context.Context) {
	// 设置HTTP路由
	// 提供静态文件服务
	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	// 噪音数据API
	http.HandleFunc("/api/noise-data", func(w http.ResponseWriter, r *http.Request) {
		s.dataMutex.Lock()
		defer s.dataMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.noiseData)
	})

	// 实时噪音数据API（1秒均值）
	http.HandleFunc("/api/noise-data/realtime", func(w http.ResponseWriter, r *http.Request) {
		s.dataMutex.Lock()
		defer s.dataMutex.Unlock()

		// 计算最近1秒的均值
		var avgLowFreqRatio, avgVolume, avgMaxSample float64
		if len(s.recentNoiseData) > 0 {
			for _, d := range s.recentNoiseData {
				avgLowFreqRatio += d.LowFreqRatio
				avgVolume += d.Volume
				avgMaxSample += d.MaxSample
			}
			avgLowFreqRatio /= float64(len(s.recentNoiseData))
			avgVolume /= float64(len(s.recentNoiseData))
			avgMaxSample /= float64(len(s.recentNoiseData))
		}

		// 返回实时数据
		realtimeData := map[string]interface{}{
			"low_freq_ratio": avgLowFreqRatio,
			"volume":         avgVolume,
			"max_sample":     avgMaxSample,
			"timestamp":      time.Now().Format("2006-01-02 15:04:05"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(realtimeData)
	})

	// 检测日志API
	http.HandleFunc("/api/detection-logs", func(w http.ResponseWriter, r *http.Request) {
		s.dataMutex.Lock()
		defer s.dataMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.detectionLogs)
	})

	// 播放随机音频API
	http.HandleFunc("/api/audio/play/random", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 检查是否正在播放声音
		s.playMutex.Lock()
		playing := *s.isPlaying
		s.playMutex.Unlock()

		if playing {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Audio is already playing",
			})
			return
		}

		// 在后台播放音频
		go func() {
			logger.Log("Starting audio playback goroutine")
			// 获取音频文件列表
			logger.Log("Acquiring dataMutex")
			s.dataMutex.Lock()
			logger.Log("Acquired dataMutex")
			audioFiles := s.audioFiles
			s.dataMutex.Unlock()
			logger.Log("Released dataMutex")

			logger.Log("Number of audio files: %d", len(audioFiles))
			if len(audioFiles) == 0 {
				logger.Log("No audio files found")
				return
			}

			// 随机选择一个音频文件
			randomFile := audioFiles[rand.Intn(len(audioFiles))]
			logger.Log("Playing random audio: %s", randomFile)

			// 设置播放状态为 true
			logger.Log("Acquiring playMutex")
			s.playMutex.Lock()
			logger.Log("Acquired playMutex")
			*s.isPlaying = true
			s.playMutex.Unlock()
			logger.Log("Released playMutex")
			logger.Log("Set isPlaying to true")

			// 确保播放完成后重置播放状态
			defer func() {
				logger.Log("Acquiring playMutex for reset")
				s.playMutex.Lock()
				logger.Log("Acquired playMutex for reset")
				*s.isPlaying = false
				s.playMutex.Unlock()
				logger.Log("Released playMutex for reset")
				logger.Log("Set isPlaying to false")
			}()

			// 播放选中的音频
			logger.Log("Calling player.PlayAudio")
			err := player.PlayAudio(randomFile)
			if err != nil {
				logger.Log("Error playing audio: %v", err)
			} else {
				logger.Log("Audio playback completed")
			}
		}()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Playing random audio",
		})
	})

	// 连续播放3次API
	http.HandleFunc("/api/audio/play/sequence", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 检查是否正在播放声音
		s.playMutex.Lock()
		playing := *s.isPlaying
		s.playMutex.Unlock()

		if playing {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Audio is already playing",
			})
			return
		}

		// 连续播放3次随机音频
		go func() {
			// 获取音频文件列表
			s.dataMutex.Lock()
			audioFiles := s.audioFiles
			s.dataMutex.Unlock()

			if len(audioFiles) == 0 {
				logger.Log("No audio files found")
				return
			}

			// 设置播放状态为 true
			s.playMutex.Lock()
			*s.isPlaying = true
			s.playMutex.Unlock()

			defer func() {
				s.playMutex.Lock()
				*s.isPlaying = false
				s.playMutex.Unlock()
				logger.Log("Audio sequence playback completed")
			}()

			for i := 0; i < 3; i++ {
				// 随机选择一个音频文件
				randomFile := audioFiles[rand.Intn(len(audioFiles))]
				logger.Log("Playing (%d/3): %s", i+1, randomFile)

				// 播放选中的音频
				err := player.PlayAudio(randomFile)
				if err != nil {
					logger.Log("Error playing audio: %v", err)
					// 继续播放下一个，不停止
					continue
				}
			}
		}()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Starting audio sequence playback",
		})
	})

	// 停止播放API
	http.HandleFunc("/api/audio/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 这里可以添加停止音频播放的逻辑
		// 由于player.PlayAudio是阻塞的，所以在这个API中可能无法直接停止正在播放的音频
		// 但我们可以返回成功，因为前端只是需要一个反馈
		logger.Log("Stop audio requested")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Audio stop requested",
		})
	})

	// 启动HTTP服务器
	go func() {
		addr := fmt.Sprintf(":%d", s.config.HTTPPort)
		server := &http.Server{
			Addr:    addr,
			Handler: nil, // 使用默认的ServeMux
		}

		// 启动服务器
		go func() {
			logger.Log("HTTP server started on %s", addr)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Log("Error starting HTTP server: %v", err)
			}
		}()

		// 等待退出信号
		<-ctx.Done()

		// 优雅关闭服务器
		logger.Log("Shutting down HTTP server...")
		if err := server.Shutdown(context.Background()); err != nil {
			logger.Log("Error shutting down HTTP server: %v", err)
		}
		logger.Log("HTTP server stopped")
	}()
}

// AddNoiseData 添加噪音数据
func (s *HTTPServer) AddNoiseData(data map[string]interface{}) {
	// 处理噪音数据
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	lowFreqRatio, _ := data["low_freq_ratio"].(float64)
	volume, _ := data["volume"].(float64)
	maxSample, _ := data["max_sample"].(float64)
	t, _ := data["time"].(time.Time)

	// 加锁保护噪音数据
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	// 添加新的噪音数据
	s.noiseData = append(s.noiseData, NoiseData{
		Timestamp:    timestamp,
		LowFreqRatio: lowFreqRatio,
		Volume:       volume,
		MaxSample:    maxSample,
		Time:         t,
	})

	// 限制噪音数据的大小，只保留最近的1000条
	if len(s.noiseData) > 1000 {
		s.noiseData = s.noiseData[len(s.noiseData)-1000:]
	}

	// 维护最近1秒的噪音数据
	s.recentNoiseData = append(s.recentNoiseData, NoiseData{
		Timestamp:    timestamp,
		LowFreqRatio: lowFreqRatio,
		Volume:       volume,
		MaxSample:    maxSample,
		Time:         t,
	})

	// 移除1秒前的数据
	oneSecondAgo := time.Now().Add(-1 * time.Second)
	var filteredData []NoiseData
	for _, d := range s.recentNoiseData {
		if d.Time.After(oneSecondAgo) {
			filteredData = append(filteredData, d)
		}
	}
	s.recentNoiseData = filteredData

	// 将噪音数据持久化到文件
	s.SaveNoiseDataToFile()
}

// SaveNoiseDataToFile 将噪音数据保存到文件
func (s *HTTPServer) SaveNoiseDataToFile() {
	// 注意：这里不需要再获取dataMutex锁，因为调用此函数的AddNoiseData已经持有了锁

	// 将噪音数据转换为JSON
	data, err := json.Marshal(s.noiseData)
	if err != nil {
		logger.Log("Error marshaling noise data: %v", err)
		return
	}

	// 写入文件
	err = ioutil.WriteFile(s.dataFile, data, 0644)
	if err != nil {
		logger.Log("Error writing noise data to file: %v", err)
		return
	}

}

// AddDetectionLog 添加检测日志
func (s *HTTPServer) AddDetectionLog(message string, logType string) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	s.detectionLogs = append(s.detectionLogs, DetectionLog{
		Timestamp: timestamp,
		Message:   message,
		Type:      logType,
	})

	// 限制日志数量，只保留最近100条
	if len(s.detectionLogs) > 100 {
		s.detectionLogs = s.detectionLogs[len(s.detectionLogs)-100:]
	}
}

// LoadNoiseDataFromFile 从文件加载噪音数据
func (s *HTTPServer) LoadNoiseDataFromFile() {
	// 检查文件是否存在
	if _, err := os.Stat(s.dataFile); os.IsNotExist(err) {
		logger.Log("Noise data file not found, creating new one: %s", s.dataFile)
		return
	}

	// 读取文件
	data, err := ioutil.ReadFile(s.dataFile)
	if err != nil {
		logger.Log("Error reading noise data from file: %v", err)
		return
	}

	// 解析JSON
	var noiseData []NoiseData
	err = json.Unmarshal(data, &noiseData)
	if err != nil {
		logger.Log("Error unmarshaling noise data: %v", err)
		return
	}

	// 加锁保护噪音数据
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	// 设置噪音数据
	s.noiseData = noiseData
	logger.Log("Loaded %d noise data entries from file: %s", len(noiseData), s.dataFile)
}
