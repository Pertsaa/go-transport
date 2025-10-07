package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Audio settings
const (
	SampleRate = 48000 // 48 kHz
	Channels   = 2     // Stereo
)

func main() {
	inputDir := "../audio/input"
	outputDir := "../audio/output"

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Walk through all files in inputDir
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .mp3 files
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
			inputFile := path
			outputFile := filepath.Join(
				outputDir,
				strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))+".pcm",
			)

			fmt.Printf("🎧 Converting %s → %s\n", inputFile, outputFile)

			// ffmpeg command to produce raw PCM (16-bit LE, mono, 48kHz)
			cmd := exec.Command("ffmpeg",
				"-y", // Overwrite existing output
				"-i", inputFile,
				"-f", "s16le", // Raw PCM
				"-acodec", "pcm_s16le",
				"-ar", fmt.Sprintf("%d", SampleRate), // Sample rate
				"-ac", fmt.Sprintf("%d", Channels), // Channels
				outputFile,
			)

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				log.Printf("❌ Error converting %s: %v", inputFile, err)
			} else {
				fmt.Printf("✅ Done: %s\n", outputFile)
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Error walking input directory: %v", err)
	}

	fmt.Println("🏁 All conversions complete.")
}
