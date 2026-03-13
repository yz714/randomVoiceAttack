package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"randomVoiceAttack/detector"
	"randomVoiceAttack/player"
	"randomVoiceAttack/utils"
)

type AudioAnalysisReport struct {
	Directory string                     `json:"directory"`
	Files     []AudioFileAnalysisResult `json:"files"`
}

type AudioFileAnalysisResult struct {
	Filename     string                   `json:"filename"`
	LowFreqRatio float64                  `json:"low_freq_ratio"`
	TotalEnergy  float64                  `json:"total_energy"`
	Volume       float64                  `json:"volume"`
	MaxSample    float64                  `json:"max_sample"`
	Error        string                   `json:"error,omitempty"`
}

func main() {
	var dirPath string
	var outputPath string
	var debugMode bool

	flag.StringVar(&dirPath, "dir", "res", "Directory containing audio files")
	flag.StringVar(&outputPath, "output", "audio_analysis.json", "Output JSON file path")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode")
	flag.Parse()

	detector.Debug = debugMode

	fmt.Printf("Analyzing audio files in directory: %s\n", dirPath)

	audioFiles, err := utils.GetAudioFiles(dirPath)
	if err != nil {
		fmt.Printf("Error getting audio files: %v\n", err)
		os.Exit(1)
	}

	if len(audioFiles) == 0 {
		fmt.Println("No audio files found in directory")
		os.Exit(1)
	}

	fmt.Printf("Found %d audio files\n", len(audioFiles))

	report := AudioAnalysisReport{
		Directory: dirPath,
		Files:     make([]AudioFileAnalysisResult, 0, len(audioFiles)),
	}

	for _, filePath := range audioFiles {
		filename := filepath.Base(filePath)
		fmt.Printf("Analyzing: %s... ", filename)

		result := AudioFileAnalysisResult{
			Filename: filename,
		}

		samples, err := player.ReadAudioToSamples(filePath, detector.SampleSize()*10)
		if err != nil {
			result.Error = err.Error()
			fmt.Printf("ERROR: %v\n", err)
			report.Files = append(report.Files, result)
			continue
		}

		if len(samples) > detector.SampleSize() {
			mid := len(samples) / 2
			start := mid - detector.SampleSize()/2
			if start < 0 {
				start = 0
			}
			if start+detector.SampleSize() > len(samples) {
				start = len(samples) - detector.SampleSize()
			}
			samples = samples[start : start+detector.SampleSize()]
		}

		if debugMode {
			fmt.Printf("\n[DEBUG] Read %d samples\n", len(samples))
			if len(samples) > 0 {
				fmt.Printf("[DEBUG] First 5 samples: ")
				for i := 0; i < 5 && i < len(samples); i++ {
					fmt.Printf("%.6f ", samples[i])
				}
				fmt.Println()
			}
		}

		analysis := detector.AnalyzeAudio(samples)
		result.LowFreqRatio = analysis.LowFreqRatio
		result.TotalEnergy = analysis.TotalEnergy
		result.Volume = analysis.Volume
		result.MaxSample = analysis.MaxSample

		if debugMode {
			fmt.Printf("[DEBUG] Analysis result - LowFreqRatio: %.6f, TotalEnergy: %.6f, Volume: %.6f, MaxSample: %.6f\n", 
				result.LowFreqRatio, result.TotalEnergy, result.Volume, result.MaxSample)
		}

		fmt.Printf("OK - LowFreqRatio: %.4f, Volume: %.4f\n", result.LowFreqRatio, result.Volume)
		report.Files = append(report.Files, result)
	}

	fmt.Println("\n=== Analysis Summary ===")
	fmt.Printf("Total files analyzed: %d\n", len(report.Files))
	
	var successCount, errorCount int
	var avgLowFreqRatio float64
	
	for _, f := range report.Files {
		if f.Error == "" {
			successCount++
			avgLowFreqRatio += f.LowFreqRatio
		} else {
			errorCount++
		}
	}
	
	if successCount > 0 {
		avgLowFreqRatio /= float64(successCount)
		fmt.Printf("Successful analyses: %d\n", successCount)
		fmt.Printf("Average low frequency ratio: %.4f\n", avgLowFreqRatio)
	}
	
	if errorCount > 0 {
		fmt.Printf("Failed analyses: %d\n", errorCount)
	}

	outputData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Printf("Error generating JSON output: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(outputPath, outputData, 0644)
	if err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nAnalysis report saved to: %s\n", outputPath)
}
