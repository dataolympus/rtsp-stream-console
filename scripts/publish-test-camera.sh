#!/usr/bin/env bash

set -euo pipefail

CAMERA_NAME="${1:-camera-1}"
RTSP_URL="rtsp://localhost:8554/${CAMERA_NAME}"

echo "Publishing test camera to:"
echo "${RTSP_URL}"

exec ffmpeg \
  -re \
  -f lavfi \
  -i "testsrc2=size=1280x720:rate=15" \
  -c:v libx264 \
  -preset ultrafast \
  -tune zerolatency \
  -pix_fmt yuv420p \
  -rtsp_transport tcp \
  -f rtsp \
  "${RTSP_URL}"
