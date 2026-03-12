package detector

import (
	"testing"
)

// 测试readFromMicrophoneWinMM方法
func TestReadFromMicrophoneWinMM(t *testing.T) {
	// 调用readFromMicrophoneWinMM方法
	samples, err := readFromMicrophoneWinMM()
	if err != nil {
		t.Logf("readFromMicrophoneWinMM returned error: %v", err)
		// 注意：在某些环境中，可能没有麦克风设备，所以这里不应该直接失败
		// 而是记录错误并继续测试
	}
	
	// 如果成功读取到样本，验证样本长度是否合理
	if len(samples) > 0 {
		// 验证样本值在合理范围内
		for i, sample := range samples {
			if sample < -1.0 || sample > 1.0 {
				t.Errorf("Sample %d is out of range: %f", i, sample)
			}
		}
	}
}
