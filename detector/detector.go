package detector

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gonum.org/v1/gonum/dsp/fourier"
)

// Detector 定义音频检测器接口
type Detector interface {
	DetectSound() (bool, error)
	DetectLowFrequencySound() (bool, error)
	SaveNoiseSample(samples []float64) error
	SetDetectorConfig(cfg DetectorConfig)
}

const (
	lowFrequencyMin = 20
	lowFrequencyMax = 250
	sampleRate      = 44100
	sampleSize      = 1024
	maxFileSize     = 100 * 1024 * 1024
)

type DetectorConfig struct {
	VolumeThreshold       float64
	LowFreqRatioThreshold float64
	TotalEnergyThreshold  float64
}

type DetectionLog struct {
	Message string
	Type    string
}

var (
	recordDir        = "records"
	useMicrophone    = true
	NoiseDataChan    = make(chan map[string]interface{}, 100)
	DetectionLogChan = make(chan DetectionLog, 100)
	Debug            = false
	detectorConfig   = DetectorConfig{
		VolumeThreshold:       0.005,
		LowFreqRatioThreshold: 0.015,
		TotalEnergyThreshold:  0.01,
	}
	configMutex sync.RWMutex
	droppedNoiseDataCount uint64
	droppedLogCount       uint64
)

func SetDetectorConfig(cfg DetectorConfig) {
	configMutex.Lock()
	defer configMutex.Unlock()
	detectorConfig = cfg
}

func getDetectorConfig() DetectorConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return detectorConfig
}

func calculateVolumeAndMax(samples []float64) (float64, float64) {
	var sum float64
	var maxSample float64
	for _, sample := range samples {
		sum += sample * sample
		if math.Abs(sample) > maxSample {
			maxSample = math.Abs(sample)
		}
	}
	volume := math.Sqrt(sum / float64(len(samples)))
	return volume, maxSample
}

func getSamples() ([]float64, error) {
	var samples []float64
	var err error

	if useMicrophone {
		samples, err = readFromMicrophoneWinMM()
		if err != nil {
			if Debug {
				fmt.Printf("Error reading from microphone: %v, using test samples instead\n", err)
			}
			samples = generateTestSamples()
		} else {
			if Debug {
				fmt.Printf("Read %d samples from microphone\n", len(samples))
			}
			if len(samples) > 0 {
				volume, maxSample := calculateVolumeAndMax(samples)
				if Debug {
					fmt.Printf("Microphone volume: %f, Max sample: %f\n", volume, maxSample)
					fmt.Print("First 10 samples: ")
					for i := 0; i < 10 && i < len(samples); i++ {
						fmt.Printf("%f ", samples[i])
					}
					fmt.Println()
				}

				noiseData := map[string]interface{}{
					"volume":         volume,
					"max_sample":     maxSample,
					"time":           time.Now(),
					"low_freq_ratio": 0.0,
				}
				select {
				case NoiseDataChan <- noiseData:
				default:
					droppedNoiseDataCount++
					if Debug {
						fmt.Printf("Warning: Noise data channel full, dropped %d entries\n", droppedNoiseDataCount)
					}
				}
			}
		}
	} else {
		if Debug {
			fmt.Println("Using test samples")
		}
		samples = generateTestSamples()
	}

	return samples, nil
}

func DetectSound() (bool, error) {
	return DetectLowFrequencySound()
}

func DetectLowFrequencySound() (bool, error) {
	samples, err := getSamples()
	if err != nil {
		return false, err
	}

	isLowFreq := detectLowFrequency(samples)
	if isLowFreq {
		SaveNoiseSample(samples)
	}
	return isLowFreq, nil
}

func generateTestSamples() []float64 {
	samples := make([]float64, sampleSize)
	for i := range samples {
		samples[i] += 0.1 * (rand.Float64()*2 - 1)
		if samples[i] > 1.0 {
			samples[i] = 1.0
		} else if samples[i] < -1.0 {
			samples[i] = -1.0
		}
	}
	return samples
}

func detectLowFrequency(samples []float64) bool {
	volume, maxSample := calculateVolumeAndMax(samples)

	fft := fourier.NewFFT(len(samples))
	spectrum := fft.Coefficients(nil, samples)

	freqResolution := float64(sampleRate) / float64(len(samples))

	var lowFreqEnergy float64
	var totalEnergy float64

	for i, coeff := range spectrum {
		freq := float64(i) * freqResolution
		energy := real(coeff)*real(coeff) + imag(coeff)*imag(coeff)
		totalEnergy += energy

		if freq >= lowFrequencyMin && freq <= lowFrequencyMax {
			lowFreqEnergy += energy
		}
	}

	cfg := getDetectorConfig()
	if totalEnergy > cfg.TotalEnergyThreshold {
		lowFreqRatio := lowFreqEnergy / totalEnergy
		if Debug {
			fmt.Printf("Low frequency ratio: %f, Total energy: %f, Volume: %f, Max sample: %f\n", lowFreqRatio, totalEnergy, volume, maxSample)
		}

		noiseData := map[string]interface{}{
			"low_freq_ratio": lowFreqRatio,
			"time":           time.Now(),
			"volume":         volume,
			"max_sample":     maxSample,
		}
		select {
		case NoiseDataChan <- noiseData:
		default:
			droppedNoiseDataCount++
			if Debug {
				fmt.Printf("Warning: Noise data channel full, dropped %d entries\n", droppedNoiseDataCount)
			}
		}

		detected := volume > cfg.VolumeThreshold && lowFreqRatio > cfg.LowFreqRatioThreshold
		if detected {
			select {
			case DetectionLogChan <- DetectionLog{
				Message: fmt.Sprintf("Low frequency detected! Ratio: %.4f, Volume: %.4f", lowFreqRatio, volume),
				Type:    "detection",
			}:
			default:
				droppedLogCount++
				if Debug {
					fmt.Printf("Warning: Detection log channel full, dropped %d entries\n", droppedLogCount)
				}
			}
		}
		return detected
	}

	if Debug {
		fmt.Println("Total energy is too low or zero")
	}
	return false
}

func SaveNoiseSample(samples []float64) error {
	if err := os.MkdirAll(recordDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(recordDir, fmt.Sprintf("noise_%s.wav", timestamp))

	totalSize, err := getDirSize(recordDir)
	if err != nil {
		return err
	}

	if totalSize >= maxFileSize {
		if err := deleteOldestFile(recordDir); err != nil {
			return err
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	numSamples := len(samples)
	sampleRateVal := 44100
	numChannels := 1
	bitsPerSample := 16
	dataSize := numSamples * numChannels * bitsPerSample / 8
	fileSize := 44 + dataSize

	_, err = file.Write([]byte("RIFF"))
	if err != nil {
		return err
	}

	_, err = file.Write([]byte{byte(fileSize & 0xFF), byte((fileSize >> 8) & 0xFF), byte((fileSize >> 16) & 0xFF), byte((fileSize >> 24) & 0xFF)})
	if err != nil {
		return err
	}

	_, err = file.Write([]byte("WAVE"))
	if err != nil {
		return err
	}

	_, err = file.Write([]byte("fmt "))
	if err != nil {
		return err
	}

	_, err = file.Write([]byte{16, 0, 0, 0})
	if err != nil {
		return err
	}

	_, err = file.Write([]byte{1, 0})
	if err != nil {
		return err
	}

	_, err = file.Write([]byte{byte(numChannels), 0})
	if err != nil {
		return err
	}

	_, err = file.Write([]byte{byte(sampleRateVal & 0xFF), byte((sampleRateVal >> 8) & 0xFF), byte((sampleRateVal >> 16) & 0xFF), byte((sampleRateVal >> 24) & 0xFF)})
	if err != nil {
		return err
	}

	byteRate := sampleRateVal * numChannels * bitsPerSample / 8
	_, err = file.Write([]byte{byte(byteRate & 0xFF), byte((byteRate >> 8) & 0xFF), byte((byteRate >> 16) & 0xFF), byte((byteRate >> 24) & 0xFF)})
	if err != nil {
		return err
	}

	blockAlign := numChannels * bitsPerSample / 8
	_, err = file.Write([]byte{byte(blockAlign), 0})
	if err != nil {
		return err
	}

	_, err = file.Write([]byte{byte(bitsPerSample), 0})
	if err != nil {
		return err
	}

	_, err = file.Write([]byte("data"))
	if err != nil {
		return err
	}

	_, err = file.Write([]byte{byte(dataSize & 0xFF), byte((dataSize >> 8) & 0xFF), byte((dataSize >> 16) & 0xFF), byte((dataSize >> 24) & 0xFF)})
	if err != nil {
		return err
	}

	for _, sample := range samples {
		sampleInt16 := int16(sample * 32767)
		_, err = file.Write([]byte{byte(sampleInt16 & 0xFF), byte((sampleInt16 >> 8) & 0xFF)})
		if err != nil {
			return err
		}
	}

	return nil
}

func getDirSize(dir string) (int64, error) {
	var size int64

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

func deleteOldestFile(dir string) error {
	var oldestFile string
	var oldestTime time.Time

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			if oldestFile == "" || file.ModTime().Before(oldestTime) {
				oldestFile = filepath.Join(dir, file.Name())
				oldestTime = file.ModTime()
			}
		}
	}

	if oldestFile != "" {
		return os.Remove(oldestFile)
	}

	return nil
}

func TestSaveNoise() error {
	samples := generateTestSamples()
	return SaveNoiseSample(samples)
}
