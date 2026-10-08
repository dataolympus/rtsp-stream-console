package stream

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func TestFFmpegRunnerRealRTSP(t *testing.T) {
	sourceURL := os.Getenv("RTSP_INTEGRATION_URL")
	if sourceURL == "" {
		t.Skip("RTSP_INTEGRATION_URL is not set")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	runner := NewFFmpegRunner("ffmpeg")

	runtime, err := runner.Start(
		ctx,
		sourceURL,
	)
	if err != nil {
		t.Fatalf("start FFmpeg runner: %v", err)
	}

	packet := make([]byte, 188)

	if _, err := io.ReadFull(
		runtime.Output,
		packet,
	); err != nil {
		t.Fatalf(
			"read MPEG-TS packet: %v",
			err,
		)
	}

	if packet[0] != 0x47 {
		t.Fatalf(
			"expected MPEG-TS sync byte 0x47, got 0x%02x",
			packet[0],
		)
	}

	cancel()

	select {
	case <-runtime.Done:
	case <-time.After(2 * time.Second):
		t.Fatal("FFmpeg did not exit after cancellation")
	}
}
