package analyzer

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// DownscaleVideo pre-processes video bytes to a lightweight 360p resolution,
// reduced framerate (e.g. 2 fps), and low bitrate for optimal token economy and low memory usage.
// If ffmpeg is not installed or fails, it gracefully returns the original bytes.
func DownscaleVideo(ctx context.Context, inputBytes []byte, maxDimension int) ([]byte, error) {
	if len(inputBytes) == 0 {
		return nil, fmt.Errorf("input video bytes cannot be empty")
	}
	if maxDimension <= 0 {
		maxDimension = 360
	}

	// Check if ffmpeg binary exists in PATH
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Printf("[VideoOptimizer] ffmpeg not found in PATH; falling back to original video payload (%d bytes)", len(inputBytes))
		return inputBytes, nil
	}

	tempDir, err := os.MkdirTemp("", "reeler-video-downscale-*")
	if err != nil {
		return inputBytes, nil
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	inputFile := filepath.Join(tempDir, "input.mp4")
	outputFile := filepath.Join(tempDir, "downscaled.mp4")

	if err := os.WriteFile(inputFile, inputBytes, 0600); err != nil {
		return inputBytes, nil
	}

	// Timeout context for downscaling process (max 90 seconds)
	processCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	// Scale video to max 360p, 1 FPS sampling, H.264 CRF 34, ultrafast preset, 1 thread for low memory & CPU safety
	scaleFilter := fmt.Sprintf("scale='min(%d,iw)':-2", maxDimension)
	cmd := exec.CommandContext(processCtx, ffmpegPath,
		"-y",
		"-threads", "1",
		"-i", inputFile,
		"-vf", scaleFilter,
		"-r", "1", // 1 FPS is ideal for Gemini video understanding (Gemini internally samples 1 FPS)
		"-c:v", "libx264",
		"-crf", "34",
		"-preset", "ultrafast",
		"-an", // Strip audio track to save tokens/bandwidth
		"-movflags", "+faststart",
		outputFile,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("[VideoOptimizer] ffmpeg downscale failed: %v (stderr: %s); using raw bytes", err, stderr.String())
		return inputBytes, nil
	}

	downscaledBytes, err := os.ReadFile(outputFile)
	if err != nil || len(downscaledBytes) == 0 {
		log.Printf("[VideoOptimizer] failed reading downscaled file; using raw bytes")
		return inputBytes, nil
	}

	reductionPct := 100.0 - (float64(len(downscaledBytes)) / float64(len(inputBytes)) * 100.0)
	log.Printf("[VideoOptimizer] Successfully downscaled video from %d KB to %d KB (%.1f%% reduction)",
		len(inputBytes)/1024, len(downscaledBytes)/1024, reductionPct)

	return downscaledBytes, nil
}
