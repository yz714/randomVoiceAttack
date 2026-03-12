package detector

import (
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 测试DetectSound方法
func TestDetectSound(t *testing.T) {
	// 调用DetectSound方法
	result, err := DetectSound()
	if err != nil {
		t.Errorf("DetectSound returned error: %v", err)
	}
	
	// 验证结果是布尔值
	_ = result
}

// 测试DetectLowFrequencySound方法
func TestDetectLowFrequencySound(t *testing.T) {
	// 保存原始的useMicrophone值
	originalUseMicrophone := useMicrophone
	// 暂时设置为false，使用测试数据
	useMicrophone = false
	defer func() {
		// 恢复原始值
		useMicrophone = originalUseMicrophone
	}()
	
	// 调用DetectLowFrequencySound方法
	result, err := DetectLowFrequencySound()
	if err != nil {
		t.Errorf("DetectLowFrequencySound returned error: %v", err)
	}
	
	// 验证结果是布尔值
	_ = result
}

// 测试generateTestSamples方法
func TestGenerateTestSamples(t *testing.T) {
	// 调用generateTestSamples方法
	samples := generateTestSamples()
	
	// 验证返回的样本长度正确
	if len(samples) != sampleSize {
		t.Errorf("Expected %d samples, got %d", sampleSize, len(samples))
	}
	
	// 验证样本值在合理范围内
	for i, sample := range samples {
		if sample < -1.0 || sample > 1.0 {
			t.Errorf("Sample %d is out of range: %f", i, sample)
		}
	}
}

// 测试detectLowFrequency方法
func TestDetectLowFrequency(t *testing.T) {
	// 生成包含低频成分的测试样本
	lowFreqSamples := make([]float64, sampleSize)
	for i := range lowFreqSamples {
		// 加入20Hz的低频成分
		lowFreqSamples[i] = 0.8 * math.Sin(2*math.Pi*20*float64(i)/sampleRate)
	}
	
	// 测试低频样本
	result := detectLowFrequency(lowFreqSamples)
	if !result {
		t.Error("Expected to detect low frequency in low frequency samples")
	}
	
	// 生成包含高频成分的测试样本
	highFreqSamples := make([]float64, sampleSize)
	for i := range highFreqSamples {
		// 加入1000Hz的高频成分
		highFreqSamples[i] = 0.8 * math.Sin(2*math.Pi*1000*float64(i)/sampleRate)
	}
	
	// 测试高频样本
	result = detectLowFrequency(highFreqSamples)
	if result {
		t.Error("Expected not to detect low frequency in high frequency samples")
	}
}

// 测试SaveNoiseSample方法
func TestSaveNoiseSample(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := ioutil.TempDir("", "detector_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// 保存原始的recordDir值
	originalRecordDir := recordDir
	// 暂时设置为临时目录
	recordDir = tempDir
	defer func() {
		// 恢复原始值
		recordDir = originalRecordDir
	}()
	
	// 生成测试样本
	samples := generateTestSamples()
	
	// 调用SaveNoiseSample方法
	err = SaveNoiseSample(samples)
	if err != nil {
		t.Errorf("SaveNoiseSample returned error: %v", err)
	}
	
	// 验证文件是否创建成功
	files, err := ioutil.ReadDir(tempDir)
	if err != nil {
		t.Errorf("Failed to read temp dir: %v", err)
	}
	
	if len(files) == 0 {
		t.Error("No file was created")
	}
	
	// 验证文件是否为WAV格式
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".wav" {
			t.Errorf("Expected WAV file, got %s", file.Name())
		}
	}
}

// 测试getDirSize方法
func TestGetDirSize(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := ioutil.TempDir("", "detector_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// 创建一个测试文件
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, world!"
	err = ioutil.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// 调用getDirSize方法
	size, err := getDirSize(tempDir)
	if err != nil {
		t.Errorf("getDirSize returned error: %v", err)
	}
	
	// 验证目录大小正确
	expectedSize := int64(len(testContent))
	if size != expectedSize {
		t.Errorf("Expected directory size %d, got %d", expectedSize, size)
	}
}

// 测试deleteOldestFile方法
func TestDeleteOldestFile(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := ioutil.TempDir("", "detector_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// 创建多个测试文件，带有不同的修改时间
	testFiles := []string{"old.txt", "middle.txt", "new.txt"}
	for _, filename := range testFiles {
		testFile := filepath.Join(tempDir, filename)
		testContent := filename
		err = ioutil.WriteFile(testFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		
		// 为了确保文件有不同的修改时间
		time.Sleep(10 * time.Millisecond)
	}
	
	// 调用deleteOldestFile方法
	err = deleteOldestFile(tempDir)
	if err != nil {
		t.Errorf("deleteOldestFile returned error: %v", err)
	}
	
	// 验证最旧的文件是否被删除
	files, err := ioutil.ReadDir(tempDir)
	if err != nil {
		t.Errorf("Failed to read temp dir: %v", err)
	}
	
	// 应该剩下2个文件
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
	
	// 验证old.txt是否被删除
	fileNames := make(map[string]bool)
	for _, file := range files {
		fileNames[file.Name()] = true
	}
	
	if fileNames["old.txt"] {
		t.Error("old.txt should have been deleted")
	}
	
	if !fileNames["middle.txt"] || !fileNames["new.txt"] {
		t.Error("middle.txt and new.txt should still exist")
	}
}

// 测试TestSaveNoise方法
func TestTestSaveNoise(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := ioutil.TempDir("", "detector_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// 保存原始的recordDir值
	originalRecordDir := recordDir
	// 暂时设置为临时目录
	recordDir = tempDir
	defer func() {
		// 恢复原始值
		recordDir = originalRecordDir
	}()
	
	// 调用TestSaveNoise方法
	err = TestSaveNoise()
	if err != nil {
		t.Errorf("TestSaveNoise returned error: %v", err)
	}
	
	// 验证文件是否创建成功
	files, err := ioutil.ReadDir(tempDir)
	if err != nil {
		t.Errorf("Failed to read temp dir: %v", err)
	}
	
	if len(files) == 0 {
		t.Error("No file was created")
	}
}
