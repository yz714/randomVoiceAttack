package detector

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

const (
	WAVE_FORMAT_PCM = 1
)

type WAVEFORMATEX struct {
	wFormatTag      uint16
	nChannels       uint16
	nSamplesPerSec  uint32
	nAvgBytesPerSec uint32
	nBlockAlign     uint16
	wBitsPerSample  uint16
	cbSize          uint16
}

type HWAVEOUT uintptr

type HWAVEIN uintptr

var (
	winmm = syscall.NewLazyDLL("winmm.dll")

	waveInOpen            = winmm.NewProc("waveInOpen")
	waveInClose           = winmm.NewProc("waveInClose")
	waveInStart           = winmm.NewProc("waveInStart")
	waveInStop            = winmm.NewProc("waveInStop")
	waveInReset           = winmm.NewProc("waveInReset")
	waveInPrepareHeader   = winmm.NewProc("waveInPrepareHeader")
	waveInUnprepareHeader = winmm.NewProc("waveInUnprepareHeader")
	waveInAddBuffer       = winmm.NewProc("waveInAddBuffer")
)

type WAVEHDR struct {
	lpData          uintptr
	dwBufferLength  uint32
	dwBytesRecorded uint32
	dwUser          uintptr
	dwFlags         uint32
	dwLoops         uint32
	lpNext          uintptr
	reserved        uintptr
}

// 保持音频设备打开的全局变量
var globalWaveIn HWAVEIN
var isDeviceOpen bool = false

// 从麦克风读取音频数据
func readFromMicrophoneWinMM() ([]float64, error) {
	// 定义音频格式
	format := WAVEFORMATEX{
		wFormatTag:      WAVE_FORMAT_PCM,
		nChannels:       1,
		nSamplesPerSec:  uint32(sampleRate),
		nAvgBytesPerSec: uint32(sampleRate * 2), // 16-bit PCM
		nBlockAlign:     2,
		wBitsPerSample:  16,
		cbSize:          0,
	}

	// 如果设备还没打开，打开它
	if !isDeviceOpen {
		result, _, err := waveInOpen.Call(
			uintptr(unsafe.Pointer(&globalWaveIn)),
			0, // 默认设备
			uintptr(unsafe.Pointer(&format)),
			0,
			0,
			0x00020000, // CALLBACK_NULL
		)

		if result != 0 {
			return nil, fmt.Errorf("waveInOpen failed: %v, result: %d", err, result)
		}
		isDeviceOpen = true
	}

	// 分配缓冲区 - 每次都创建新的缓冲区
	bufferSize := uint32(sampleRate * 2) // 1秒的缓冲区
	buffer := make([]byte, bufferSize)
	// 清空缓冲区，确保没有残留数据
	for i := range buffer {
		buffer[i] = 0
	}
	hdr := WAVEHDR{
		lpData:          uintptr(unsafe.Pointer(&buffer[0])),
		dwBufferLength:  bufferSize,
		dwBytesRecorded: 0,
		dwFlags:         0,
	}

	// 准备缓冲区
	result, _, err := waveInPrepareHeader.Call(uintptr(globalWaveIn), uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Sizeof(hdr)))
	if result != 0 {
		return nil, fmt.Errorf("waveInPrepareHeader failed: %v, result: %d", err, result)
	}

	// 添加缓冲区
	result, _, err = waveInAddBuffer.Call(uintptr(globalWaveIn), uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Sizeof(hdr)))
	if result != 0 {
		waveInUnprepareHeader.Call(uintptr(globalWaveIn), uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Sizeof(hdr)))
		return nil, fmt.Errorf("waveInAddBuffer failed: %v, result: %d", err, result)
	}

	// 开始录音
	result, _, err = waveInStart.Call(uintptr(globalWaveIn))
	if result != 0 {
		waveInReset.Call(uintptr(globalWaveIn))
		waveInUnprepareHeader.Call(uintptr(globalWaveIn), uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Sizeof(hdr)))
		return nil, fmt.Errorf("waveInStart failed: %v, result: %d", err, result)
	}

	// 固定时长录音 - 确保能够录制到足够的数据
	recordingDuration := 100 * time.Millisecond
	time.Sleep(recordingDuration)

	// 停止录音
	result, _, err = waveInStop.Call(uintptr(globalWaveIn))
	if result != 0 {
		// 即使停止失败，也继续处理
	}

	// 重置录音设备
	result, _, err = waveInReset.Call(uintptr(globalWaveIn))
	if result != 0 {
		// 即使重置失败，也继续处理
	}

	// 打印实际录制的字节数
	if Debug {
		fmt.Printf("Bytes recorded: %d\n", hdr.dwBytesRecorded)
	}

	// 将16-bit PCM数据转换为float64
	var samples []float64
	if hdr.dwBytesRecorded > 0 {
		// 只取前sampleSize个样本，或者所有可用样本（取较小者）
		numSamples := int(hdr.dwBytesRecorded / 2)
		if numSamples > sampleSize {
			numSamples = sampleSize
		}
		samples = make([]float64, numSamples)
		// 音量增益倍数
		gain := 500.0
		for i := 0; i < numSamples*2; i += 2 {
			// 小端格式
			sample := int16(buffer[i]) | int16(buffer[i+1])<<8
			// 转换为[-1.0, 1.0]的float64并应用增益
			samples[i/2] = float64(sample) / 32768.0 * gain
			// 确保值在[-1.0, 1.0]范围内
			if samples[i/2] > 1.0 {
				samples[i/2] = 1.0
			} else if samples[i/2] < -1.0 {
				samples[i/2] = -1.0
			}
		}
	} else {
		// 如果没有录制到数据，返回空数组
		samples = make([]float64, 0)
	}

	// 取消准备缓冲区
	waveInUnprepareHeader.Call(uintptr(globalWaveIn), uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Sizeof(hdr)))

	return samples, nil
}
