package services

import (
	"context"
	"syscall"
	"time"

	"randomVoiceAttack/logger"
	"randomVoiceAttack/player"
	"randomVoiceAttack/utils"
)

// HeartbeatConfig 心跳服务配置
type HeartbeatConfig struct {
	BluetoothHeartbeatInterval int
	AntiLockInterval           int
}

// HeartbeatService 心跳服务结构体
type HeartbeatService struct {
	config HeartbeatConfig
}

// NewHeartbeatService 创建新的心跳服务
func NewHeartbeatService(config HeartbeatConfig) *HeartbeatService {
	return &HeartbeatService{
		config: config,
	}
}

// Start 启动心跳服务
func (s *HeartbeatService) Start(ctx context.Context) {
	// 启动蓝牙心跳协程
	utils.GoWithName("bluetoothHeartbeat", func() {
		s.bluetoothHeartbeat(ctx)
	})
	// 启动防止锁屏协程
	utils.GoWithName("preventLockScreen", func() {
		s.preventLockScreen(ctx)
	})
}

// bluetoothHeartbeat 蓝牙心跳协程
func (s *HeartbeatService) bluetoothHeartbeat(ctx context.Context) {
	// 每隔指定秒数发送一次心跳，防止蓝牙断开
	ticker := time.NewTicker(time.Duration(s.config.BluetoothHeartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 收到退出信号
			logger.Log("Bluetooth heartbeat stopped")
			return
		case <-ticker.C:
			// 播放一个简短的静音音频来保持蓝牙连接
			logger.Log("Sending Bluetooth heartbeat...")
			err := player.PlaySilentAudio()
			if err != nil {
				logger.Log("Error sending Bluetooth heartbeat: %v", err)
			}
		}
	}
}

// preventLockScreen 防止Windows锁屏的协程
func (s *HeartbeatService) preventLockScreen(ctx context.Context) {
	// 导入Windows API
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setThreadExecutionState := kernel32.NewProc("SetThreadExecutionState")

	// 定义常量
	const (
		ES_CONTINUOUS       = 0x80000000
		ES_SYSTEM_REQUIRED  = 0x00000001
		ES_DISPLAY_REQUIRED = 0x00000002
	)

	// 每隔指定秒数调用一次SetThreadExecutionState
	ticker := time.NewTicker(time.Duration(s.config.AntiLockInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 收到退出信号
			logger.Log("Anti-lock screen service stopped")
			return
		case <-ticker.C:
			// 调用SetThreadExecutionState，防止系统锁屏和睡眠
			_, _, _ = setThreadExecutionState.Call(
				ES_CONTINUOUS | ES_SYSTEM_REQUIRED | ES_DISPLAY_REQUIRED,
			)
			logger.Log("Preventing screen lock...")
		}
	}
}
