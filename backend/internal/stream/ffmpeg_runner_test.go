package stream

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func writeFakeExecutable(
	t *testing.T,
	body string,
) string {
	t.Helper()

	path := filepath.Join(
		t.TempDir(),
		"fake-ffmpeg",
	)

	content := "#!/bin/sh\nset -eu\n" + body + "\n"

	if err := os.WriteFile(
		path,
		[]byte(content),
		0o755,
	); err != nil {
		t.Fatalf("write fake executable: %v", err)
	}

	return path
}

func TestFFmpegRunnerArgs(t *testing.T) {
	runner := NewFFmpegRunner("ffmpeg")

	got := runner.args(
		"rtsp://localhost:8554/camera-1",
	)

	want := []string{
		"-hide_banner",
		"-loglevel",
		"warning",
		"-rtsp_transport",
		"tcp",
		"-i",
		"rtsp://localhost:8554/camera-1",
		"-an",
		"-c:v",
		"libx264",
		"-preset",
		"ultrafast",
		"-tune",
		"zerolatency",
		"-pix_fmt",
		"yuv420p",
		"-f",
		"mpegts",
		"pipe:1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"unexpected FFmpeg args:\nwant: %#v\ngot:  %#v",
			want,
			got,
		)
	}
}

func TestFFmpegRunnerStartExposesProcessOutput(t *testing.T) {
	binary := writeFakeExecutable(
		t,
		`printf 'media-data'`,
	)

	runner := NewFFmpegRunner(binary)

	runtime, err := runner.Start(
		context.Background(),
		"rtsp://localhost:8554/camera-1",
	)
	if err != nil {
		t.Fatalf("start runner: %v", err)
	}

	got, err := io.ReadAll(runtime.Output)
	if err != nil {
		t.Fatalf("read runtime output: %v", err)
	}

	if string(got) != "media-data" {
		t.Fatalf(
			"expected output %q, got %q",
			"media-data",
			string(got),
		)
	}

	if err := <-runtime.Done; err != nil {
		t.Fatalf("expected clean process exit, got %v", err)
	}
}

func TestFFmpegRunnerReportsProcessFailure(t *testing.T) {
	binary := writeFakeExecutable(
		t,
		`
echo "simulated ffmpeg failure" >&2
exit 7
`,
	)

	runner := NewFFmpegRunner(binary)

	runtime, err := runner.Start(
		context.Background(),
		"rtsp://localhost:8554/camera-1",
	)
	if err != nil {
		t.Fatalf("start runner: %v", err)
	}

	err = <-runtime.Done
	if err == nil {
		t.Fatal("expected process failure")
	}

	if !strings.Contains(
		err.Error(),
		"simulated ffmpeg failure",
	) {
		t.Fatalf(
			"expected stderr in error, got %v",
			err,
		)
	}
}

func TestFFmpegRunnerStartFailsWhenBinaryMissing(t *testing.T) {
	runner := NewFFmpegRunner(
		filepath.Join(
			t.TempDir(),
			"missing-ffmpeg",
		),
	)

	runtime, err := runner.Start(
		context.Background(),
		"rtsp://localhost:8554/camera-1",
	)

	if err == nil {
		t.Fatal("expected start error")
	}

	if runtime != nil {
		t.Fatal("expected nil runtime")
	}
}

func TestFFmpegRunnerCancellationStopsProcess(t *testing.T) {
	binary := writeFakeExecutable(
		t,
		`exec sleep 30`,
	)

	runner := NewFFmpegRunner(binary)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	runtime, err := runner.Start(
		ctx,
		"rtsp://localhost:8554/camera-1",
	)
	if err != nil {
		t.Fatalf("start runner: %v", err)
	}

	cancel()

	select {
	case err := <-runtime.Done:
		if err == nil {
			t.Fatal("expected cancellation to terminate process")
		}

	case <-time.After(2 * time.Second):
		t.Fatal("process did not stop after cancellation")
	}
}
