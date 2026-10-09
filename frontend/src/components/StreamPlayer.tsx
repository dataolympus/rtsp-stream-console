import {
    useEffect,
    useRef,
} from 'react';

import mpegts from 'mpegts.js';

type StreamPlayerProps = {
    streamId: string;
    streamName: string;
    isLive: boolean;
};

function getWebSocketURL(
    streamId: string,
): string {
    const protocol =
        window.location.protocol === 'https:'
            ? 'wss:'
            : 'ws:';

    return (
        `${protocol}//${window.location.host}` +
        `/api/v1/streams/${streamId}/ws`
    );
}

function StreamPlayer({
    streamId,
    streamName,
    isLive,
}: StreamPlayerProps) {
    const videoRef =
        useRef<HTMLVideoElement>(null);

    useEffect(() => {
        if (!isLive) {
            return;
        }

        const video = videoRef.current;

        if (!video) {
            return;
        }

        const features =
            mpegts.getFeatureList();

        if (!features.mseLivePlayback) {
            return;
        }

        const player = mpegts.createPlayer({
            type: 'mse',
            isLive: true,
            url: getWebSocketURL(streamId),
            hasAudio: false,
            hasVideo: true,
        });

        player.attachMediaElement(video);
        player.load();

        void player.play();

        return () => {
            player.pause();
            player.unload();
            player.detachMediaElement();
            player.destroy();
        };
    }, [
        isLive,
        streamId,
    ]);

    return (
        <video
            ref={videoRef}
            aria-label={`${streamName} video`}
            muted
            autoPlay
            playsInline
            controls
        />
    );
}

export default StreamPlayer;