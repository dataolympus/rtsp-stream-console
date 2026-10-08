package stream

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type FFmpegRunner struct {
	binary string
}

func NewFFmpegRunner(binary string) *FFmpegRunner {
	return &FFmpegRunner{
		binary: binary,
	}
}

func (r *FFmpegRunner) args(sourceURL string) []string {
	return []string{
		"-hide_banner",
		"-loglevel",
		"warning",
		"-rtsp_transport",
		"tcp",
		"-i",
		sourceURL,
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
}

func (r *FFmpegRunner) Start(
	ctx context.Context,
	sourceURL string,
) (*Runtime, error) {
	cmd := exec.CommandContext(
		ctx,
		r.binary,
		r.args(sourceURL)...,
	)

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf(
			"create ffmpeg stdout pipe: %w",
			err,
		)
	}

	cmd.Stdout = stdoutWriter

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()

		return nil, fmt.Errorf(
			"start ffmpeg: %w",
			err,
		)
	}

	// The child process now owns its inherited stdout descriptor.
	// Close the parent's writer copy so the reader receives EOF
	// when the child exits.
	_ = stdoutWriter.Close()

	done := make(chan error, 1)

	go func() {
		err := cmd.Wait()

		if err != nil {
			message := strings.TrimSpace(
				stderr.String(),
			)

			if message != "" {
				err = fmt.Errorf(
					"ffmpeg exited: %w: %s",
					err,
					message,
				)
			}
		}

		done <- err
		close(done)
	}()

	return &Runtime{
		Output: stdoutReader,
		Done:   done,
	}, nil
}
