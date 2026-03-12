package detector

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	WAVE_FORMAT_PCM       = 1
	defaultRecordingDuration = 100 * time.Millisecond
	defaultGain             = 500.0
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

var (
	waveInHandle HWAVEIN
	deviceOpen   bool = false
	deviceMutex  sync.Mutex
)

// 从麦克风读取音频数据
func readFromMicrophoneWinMM() ([]float64, error) {
	deviceMutex.Lock()
	defer deviceMutex.Unlock()

	format := WAVEFORMATEX{
		wFormatTag:      WAVE_FORMAT_PCM,
		nChannels:       1,
		nSamplesPerSec:  uint32(sampleRate),
		nAvgBytesPerSec: uint32(sampleRate * 2),
		nBlockAlign:     2,
		wBitsPerSample:  16,
		cbSize:          0,
	}

	if !deviceOpen {
		result, _, err := waveInOpen.Call(
			uintptr(unsafe.Pointer(&waveInHandle)),
			0,
			uintptr(unsafe.Pointer(&format)),
			0,
			0,
			0x00020000,
		)

		if result != 0 {
			return nil, fmt.Errorf("waveInOpen failed: %v, result: %d", err, result)
		}
		deviceOpen = true
	}

	bufferSize := uint32(sampleRate * 2)
	buffer := make([]byte, bufferSize)
	for i := range buffer {
		buffer[i] = 0
	}
	hdr := WAVEHDR{
		lpData:          uintptr(unsafe.Pointer(&buffer[0])),
		dwBufferLength:  bufferSize,
		dwBytesRecorded: 0,
		dwFlags:         0,
	}

	result, _, err := waveInPrepareHeader.Call(uintptr(waveInHandle), uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Sizeof(hdr)))
	if result != 0 {
		return nil, fmt.Errorf("waveInPrepareHeader failed: %v, result: %d", err, result)
	}

	result, _, err = waveInAddBuffer.Call(uintptr(waveInHandle), uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Sizeof(hdr)))
	if result != 0 {
		waveInUnprepareHeader.Call(uintptr(waveInHandle), uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Sizeof(hdr)))
		return nil, fmt.Errorf("waveInAddBuffer failed: %v, result: %d", err, result)
	}

	result, _, err = waveInStart.Call(uintptr(waveInHandle))
	if result != 0 {
		waveInReset.Call(uintptr(waveInHandle))
		waveInUnprepareHeader.Call(uintptr(waveInHandle), uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Sizeof(hdr)))
		return nil, fmt.Errorf("waveInStart failed: %v, result: %d", err, result)
	}

	time.Sleep(defaultRecordingDuration)

	result, _, err = waveInStop.Call(uintptr(waveInHandle))
	if result != 0 && Debug {
		fmt.Printf("Warning: waveInStop failed: %v, result: %d\n", err, result)
	}

	result, _, err = waveInReset.Call(uintptr(waveInHandle))
	if result != 0 && Debug {
		fmt.Printf("Warning: waveInReset failed: %v, result: %d\n", err, result)
	}

	if Debug {
		fmt.Printf("Bytes recorded: %d\n", hdr.dwBytesRecorded)
	}

	var samples []float64
	if hdr.dwBytesRecorded > 0 {
		numSamples := int(hdr.dwBytesRecorded / 2)
		if numSamples > sampleSize {
			numSamples = sampleSize
		}
		samples = make([]float64, numSamples)
		gain := defaultGain
		for i := 0; i < numSamples*2; i += 2 {
			sample := int16(buffer[i]) | int16(buffer[i+1])<<8
			samples[i/2] = float64(sample) / 32768.0 * gain
			if samples[i/2] > 1.0 {
				samples[i/2] = 1.0
			} else if samples[i/2] < -1.0 {
				samples[i/2] = -1.0
			}
		}
	} else {
		samples = make([]float64, 0)
	}

	waveInUnprepareHeader.Call(uintptr(waveInHandle), uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Sizeof(hdr)))

	return samples, nil
}

func CloseAudioDevice() error {
	deviceMutex.Lock()
	defer deviceMutex.Unlock()

	if !deviceOpen {
		return nil
	}

	result, _, err := waveInClose.Call(uintptr(waveInHandle))
	if result != 0 {
		return fmt.Errorf("waveInClose failed: %v, result: %d", err, result)
	}

	deviceOpen = false
	waveInHandle = 0
	return nil
}
