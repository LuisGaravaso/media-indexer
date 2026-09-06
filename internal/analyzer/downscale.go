package analyzer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// DownscaleVideoReader streams video bytes from an io.Reader to a temp file, then runs
// optimized ffmpeg transcoding to 360p @ 1 FPS using ultrafast settings to minimize memory and time.
func DownscaleVideoReader(ctx context.Context, r io.Reader, maxDimension int) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("input video reader cannot be nil")
	}
	if maxDimension <= 0 {
		maxDimension = 360
	}

	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Printf("[VideoOptimizer] ffmpeg not found in PATH; reading raw stream")
		return io.ReadAll(r)
	}

	tempDir, err := os.MkdirTemp("", "reeler-video-downscale-*")
	if err != nil {
		return io.ReadAll(r)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	inputFile := filepath.Join(tempDir, "input.mp4")
	outputFile := filepath.Join(tempDir, "downscaled.mp4")

	f, err := os.Create(inputFile)
	if err != nil {
		return io.ReadAll(r)
	}

	written, err := io.Copy(f, r)
	_ = f.Close()
	if err != nil || written == 0 {
		return nil, fmt.Errorf("failed writing input video stream: %w", err)
	}

	processCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// Fast keyframe & 1 FPS downsampling, ultrafast preset, CRF 34, no audio
	scaleFilter := fmt.Sprintf("scale='min(%d,iw)':-2", maxDimension)
	cmd := exec.CommandContext(processCtx, ffmpegPath,
		"-y",
		"-i", inputFile,
		"-vf", scaleFilter,
		"-r", "1", // 1 FPS is ideal for scene narrative & dramatically faster encoding
		"-c:v", "libx264",
		"-crf", "34",
		"-preset", "ultrafast",
		"-threads", "0",
		"-an",
		"-movflags", "+faststart",
		outputFile,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("[VideoOptimizer] ffmpeg downscale failed: %v (stderr: %s); falling back to raw file", err, stderr.String())
		return os.ReadFile(inputFile)
	}

	downscaledBytes, err := os.ReadFile(outputFile)
	if err != nil || len(downscaledBytes) == 0 {
		log.Printf("[VideoOptimizer] failed reading downscaled file; falling back to raw file")
		return os.ReadFile(inputFile)
	}

	reductionPct := 100.0 - (float64(len(downscaledBytes)) / float64(written) * 100.0)
	log.Printf("[VideoOptimizer] Successfully downscaled video from %d KB to %d KB (%.1f%% reduction)",
		written/1024, len(downscaledBytes)/1024, reductionPct)

	return downscaledBytes, nil
}

// DownscaleVideo pre-processes video bytes to a lightweight 360p resolution using DownscaleVideoReader.
func DownscaleVideo(ctx context.Context, inputBytes []byte, maxDimension int) ([]byte, error) {
	if len(inputBytes) == 0 {
		return nil, fmt.Errorf("input video bytes cannot be empty")
	}
	return DownscaleVideoReader(ctx, bytes.NewReader(inputBytes), maxDimension)
}
