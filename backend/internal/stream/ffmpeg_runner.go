package stream

import (
	"bytes"
	"context"
	"fmt"
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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf(
			"create ffmpeg stdout pipe: %w",
			err,
		)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf(
			"start ffmpeg: %w",
			err,
		)
	}

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
		Output: stdout,
		Done:   done,
	}, nil
}
