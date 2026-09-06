package analyzer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownscaleVideo_EmptyInput(t *testing.T) {
	_, err := DownscaleVideo(context.Background(), nil, 360)
	assert.Error(t, err)
}

func TestDownscaleVideo_FallbackOnNonVideo(t *testing.T) {
	dummyBytes := []byte("not a real video file content")
	// Since dummyBytes isn't a valid MP4, ffmpeg will fail and it should gracefully fallback to original bytes
	out, err := DownscaleVideo(context.Background(), dummyBytes, 360)
	require.NoError(t, err)
	assert.Equal(t, dummyBytes, out)
}
