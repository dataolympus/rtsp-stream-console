import {
    render,
    screen,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
    beforeEach,
    describe,
    expect,
    it,
    vi,
} from 'vitest';

import mpegts from 'mpegts.js';

import StreamPlayer from './StreamPlayer';

vi.mock('mpegts.js', () => ({
    default: {
        getFeatureList: vi.fn(),
        createPlayer: vi.fn(),
    },
}));

const attachMediaElement = vi.fn();
const load = vi.fn();
const play = vi.fn();
const pause = vi.fn();
const unload = vi.fn();
const detachMediaElement = vi.fn();
const destroy = vi.fn();

beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(
        mpegts.getFeatureList,
    ).mockReturnValue({
        msePlayback: true,
        mseLivePlayback: true,
    } as ReturnType<
        typeof mpegts.getFeatureList
    >);

    vi.mocked(
        mpegts.createPlayer,
    ).mockReturnValue({
        attachMediaElement,
        load,
        play,
        pause,
        unload,
        detachMediaElement,
        destroy,
    } as unknown as ReturnType<
        typeof mpegts.createPlayer
    >);

    play.mockResolvedValue(undefined);
});

describe('StreamPlayer', () => {
    it('plays a live stream over WebSocket', () => {
        render(
            <StreamPlayer
                streamId="stream-1"
                streamName="Camera 1"
                isLive
            />,
        );

        const video = screen.getByLabelText(
            /camera 1 player/i,
        );

        expect(video).toBeInTheDocument();

        expect(
            mpegts.createPlayer,
        ).toHaveBeenCalledWith(
            expect.objectContaining({
                type: 'mse',
                isLive: true,
                url:
                    'ws://localhost:3000/api/v1/streams/stream-1/ws',
            }),
        );

        expect(
            attachMediaElement,
        ).toHaveBeenCalledWith(video);

        expect(load).toHaveBeenCalled();

        expect(play).toHaveBeenCalled();
    });

    it('does not create a player when the stream is not live', () => {
        render(
            <StreamPlayer
                streamId="stream-1"
                streamName="Camera 1"
                isLive={false}
            />,
        );

        expect(
            screen.getByLabelText(
                /camera 1 player/i,
            ),
        ).toBeInTheDocument();

        expect(
            mpegts.createPlayer,
        ).not.toHaveBeenCalled();
    });

    it('destroys the player when the stream stops', () => {
        const { rerender } = render(
            <StreamPlayer
                streamId="stream-1"
                streamName="Camera 1"
                isLive
            />,
        );

        expect(
            mpegts.createPlayer,
        ).toHaveBeenCalledTimes(1);

        rerender(
            <StreamPlayer
                streamId="stream-1"
                streamName="Camera 1"
                isLive={false}
            />,
        );

        expect(pause).toHaveBeenCalled();
        expect(unload).toHaveBeenCalled();
        expect(
            detachMediaElement,
        ).toHaveBeenCalled();
        expect(destroy).toHaveBeenCalled();
    });

    it('pauses and resumes the viewer without stopping the stream', async () => {
        const user = userEvent.setup();

        render(
            <StreamPlayer
                streamId="stream-1"
                streamName="Camera 1"
                isLive
            />,
        );

        expect(
            vi.mocked(mpegts.createPlayer),
        ).toHaveBeenCalledTimes(1);

        await user.click(
            screen.getByRole('button', {
                name: /pause camera 1/i,
            }),
        );

        expect(pause).toHaveBeenCalledTimes(1);
        expect(unload).toHaveBeenCalledTimes(1);
        expect(
            detachMediaElement,
        ).toHaveBeenCalledTimes(1);
        expect(destroy).toHaveBeenCalledTimes(1);

        // Pausing the viewer should destroy the existing
        // player, not create another one yet.
        expect(
            vi.mocked(mpegts.createPlayer),
        ).toHaveBeenCalledTimes(1);

        await user.click(
            screen.getByRole('button', {
                name: /play camera 1/i,
            }),
        );

        expect(
            vi.mocked(mpegts.createPlayer),
        ).toHaveBeenCalledTimes(2);

        expect(
            attachMediaElement,
        ).toHaveBeenCalledTimes(2);

        expect(load).toHaveBeenCalledTimes(2);
        expect(play).toHaveBeenCalledTimes(2);
    });
});