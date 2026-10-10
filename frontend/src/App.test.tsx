import {
    act,
    render,
    screen,
    waitFor,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
    beforeEach,
    describe,
    expect,
    it,
    vi,
} from 'vitest';

import App from './App';

let fetchMock: ReturnType<typeof vi.fn>;

beforeEach(() => {
    fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => [],
    });

    vi.stubGlobal('fetch', fetchMock);
});

vi.mock('./components/StreamPlayer', () => ({
    default: ({
        streamName,
    }: {
        streamName: string;
    }) => (
        <div
            aria-label={`${streamName} player`}
        />
    ),
}));

describe('App', () => {
    it('renders the stream console', () => {
        render(<App />);

        expect(
            screen.getByRole('heading', {
                name: /rtsp stream console/i,
            }),
        ).toBeInTheDocument();

        expect(
            screen.getByRole('button', {
                name: /add stream/i,
            }),
        ).toBeInTheDocument();
    });

    it('opens the add stream form', async () => {
        const user = userEvent.setup();

        render(<App />);

        await user.click(
            screen.getByRole('button', {
                name: /add stream/i,
            }),
        );

        expect(
            screen.getByRole('dialog'),
        ).toBeInTheDocument();

        expect(
            screen.getByLabelText(/name/i),
        ).toBeInTheDocument();

        expect(
            screen.getByLabelText(/rtsp url/i),
        ).toBeInTheDocument();

        expect(
            screen.getByRole('button', {
                name: /create stream/i,
            }),
        ).toBeInTheDocument();
    });

    it('creates a stream and shows it in the console', async () => {
        const user = userEvent.setup();

        const createdStream = {
            id: 'stream-1',
            name: 'Camera 1',
            url: 'rtsp://localhost:8554/camera-1',
            state: 'created',
            createdAt: '2026-10-09T00:00:00Z',
        };

        fetchMock
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => [],
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 201,
                json: async () => createdStream,
            });

        render(<App />);

        await user.click(
            screen.getByRole('button', {
                name: /add stream/i,
            }),
        );

        await user.type(
            screen.getByLabelText(/name/i),
            'Camera 1',
        );

        await user.type(
            screen.getByLabelText(/rtsp url/i),
            'rtsp://localhost:8554/camera-1',
        );

        await user.click(
            screen.getByRole('button', {
                name: /create stream/i,
            }),
        );

        expect(fetchMock).toHaveBeenCalledWith(
            '/api/v1/streams',
            expect.objectContaining({
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    name: 'Camera 1',
                    url: 'rtsp://localhost:8554/camera-1',
                }),
            }),
        );

        expect(
            await screen.findByText('Camera 1'),
        ).toBeInTheDocument();

        expect(
            screen.getByText(
                'rtsp://localhost:8554/camera-1',
            ),
        ).toBeInTheDocument();

        expect(
            screen.queryByRole('dialog'),
        ).not.toBeInTheDocument();
    });

    it('loads existing streams when the console opens', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            status: 200,
            json: async () => [
                {
                    id: 'stream-1',
                    name: 'Existing Camera',
                    url: 'rtsp://localhost:8554/existing-camera',
                    state: 'live',
                    createdAt: '2026-10-09T00:00:00Z',
                },
            ],
        });

        render(<App />);

        expect(
            await screen.findByText('Existing Camera'),
        ).toBeInTheDocument();

        expect(
            screen.getByText(
                'rtsp://localhost:8554/existing-camera',
            ),
        ).toBeInTheDocument();

        expect(fetchMock).toHaveBeenCalledWith(
            '/api/v1/streams',
        );
    });

    it('starts a stream and refreshes its state', async () => {
        const user = userEvent.setup();

        const createdStream = {
            id: 'stream-1',
            name: 'Camera 1',
            url: 'rtsp://localhost:8554/camera-1',
            state: 'created',
            createdAt: '2026-10-09T00:00:00Z',
        };

        const liveStream = {
            ...createdStream,
            state: 'live',
        };

        fetchMock
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => [createdStream],
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 202,
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => ({
                    ...createdStream,
                    state: 'connecting',
                }),
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => liveStream,
            });

        render(<App />);

        await screen.findByText('Camera 1');

        await user.click(
            screen.getByRole('button', {
                name: /start camera 1/i,
            }),
        );

        expect(fetchMock).toHaveBeenCalledWith(
            '/api/v1/streams/stream-1/start',
            expect.objectContaining({
                method: 'POST',
            }),
        );

        expect(
            await screen.findByText('live'),
        ).toBeInTheDocument();
    });

    it('stops a live stream and refreshes its state', async () => {
        const user = userEvent.setup();

        const liveStream = {
            id: 'stream-1',
            name: 'Camera 1',
            url: 'rtsp://localhost:8554/camera-1',
            state: 'live',
            createdAt: '2026-10-09T00:00:00Z',
        };

        const stoppedStream = {
            ...liveStream,
            state: 'stopped',
        };

        fetchMock
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => [liveStream],
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 202,
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => ({
                    ...liveStream,
                    state: 'stopping',
                }),
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => stoppedStream,
            });

        render(<App />);

        await screen.findByText('Camera 1');

        await user.click(
            screen.getByRole('button', {
                name: /stop camera 1/i,
            }),
        );

        expect(fetchMock).toHaveBeenCalledWith(
            '/api/v1/streams/stream-1/stop',
            expect.objectContaining({
                method: 'POST',
            }),
        );

        expect(
            await screen.findByText('stopped'),
        ).toBeInTheDocument();
    });

    it('shows the player for a live stream', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            status: 200,
            json: async () => [
                {
                    id: 'stream-1',
                    name: 'Camera 1',
                    url: 'rtsp://localhost:8554/camera-1',
                    state: 'live',
                    createdAt: '2026-10-09T00:00:00Z',
                },
            ],
        });

        render(<App />);

        expect(
            await screen.findByLabelText(
                /camera 1 player/i,
            ),
        ).toBeInTheDocument();
    });

    it('renders multiple streams independently', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            status: 200,
            json: async () => [
                {
                    id: 'stream-1',
                    name: 'Camera 1',
                    url: 'rtsp://localhost:8554/camera-1',
                    state: 'live',
                    createdAt: '2026-10-09T00:00:00Z',
                },
                {
                    id: 'stream-2',
                    name: 'Camera 2',
                    url: 'rtsp://localhost:8554/camera-2',
                    state: 'live',
                    createdAt: '2026-10-09T00:00:01Z',
                },
            ],
        });

        render(<App />);

        expect(
            await screen.findByText('Camera 1'),
        ).toBeInTheDocument();

        expect(
            screen.getByText('Camera 2'),
        ).toBeInTheDocument();

        expect(
            screen.getByLabelText(/camera 1 player/i),
        ).toBeInTheDocument();

        expect(
            screen.getByLabelText(/camera 2 player/i),
        ).toBeInTheDocument();

        expect(
            screen.getByRole('button', {
                name: /stop camera 1/i,
            }),
        ).toBeInTheDocument();

        expect(
            screen.getByRole('button', {
                name: /stop camera 2/i,
            }),
        ).toBeInTheDocument();
    });

    it('shows a stream failure and allows retry', async () => {
        fetchMock
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => [
                    {
                        id: 'stream-1',
                        name: 'Unreachable Camera',
                        url: 'rtsp://127.0.0.1:65534/missing',
                        state: 'error',
                        error: 'stream source became unavailable',
                        createdAt: '2026-10-09T00:00:00Z',
                    },
                ],
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 202,
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => ({
                    id: 'stream-1',
                    name: 'Unreachable Camera',
                    url: 'rtsp://127.0.0.1:65534/missing',
                    state: 'connecting',
                    createdAt: '2026-10-09T00:00:00Z',
                }),
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => ({
                    id: 'stream-1',
                    name: 'Unreachable Camera',
                    url: 'rtsp://127.0.0.1:65534/missing',
                    state: 'live',
                    createdAt: '2026-10-09T00:00:00Z',
                }),
            });

        render(<App />);

        expect(
            await screen.findByText(
                'stream source became unavailable',
            ),
        ).toBeInTheDocument();

        const retryButton = screen.getByRole(
            'button',
            {
                name: /start unreachable camera/i,
            },
        );

        await userEvent.click(retryButton);

        expect(fetchMock).toHaveBeenCalledWith(
            '/api/v1/streams/stream-1/start',
            expect.objectContaining({
                method: 'POST',
            }),
        );

        await waitFor(() => {
            expect(
                screen.queryByText(
                    'stream source became unavailable',
                ),
            ).not.toBeInTheDocument();
        });

        expect(
            await screen.findByText('live'),
        ).toBeInTheDocument();
    });

    it('shows an API error when starting a stream fails', async () => {
        const user = userEvent.setup();

        fetchMock
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => [
                    {
                        id: 'stream-1',
                        name: 'Camera 1',
                        url: 'rtsp://localhost:8554/camera-1',
                        state: 'created',
                        createdAt: '2026-10-09T00:00:00Z',
                    },
                ],
            })
            .mockResolvedValueOnce({
                ok: false,
                status: 503,
                json: async () => ({
                    error: 'stream capacity reached',
                }),
            });

        render(<App />);

        const startButton =
            await screen.findByRole(
                'button',
                {
                    name: /start camera 1/i,
                },
            );

        await user.click(startButton);

        expect(
            await screen.findByText(
                /unable to start camera 1: stream capacity reached/i,
            ),
        ).toBeInTheDocument();

        expect(
            screen.getByText('created'),
        ).toBeInTheDocument();

        expect(fetchMock).toHaveBeenCalledTimes(2);
    });

    it('shows an API error when the backend is unavailable', async () => {
        const user = userEvent.setup();

        fetchMock
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => [
                    {
                        id: 'stream-1',
                        name: 'Camera 1',
                        url: 'rtsp://localhost:8554/camera-1',
                        state: 'created',
                        createdAt: '2026-10-09T00:00:00Z',
                    },
                ],
            })
            .mockRejectedValueOnce(
                new TypeError('Failed to fetch'),
            );

        render(<App />);

        await user.click(
            await screen.findByRole(
                'button',
                {
                    name: /start camera 1/i,
                },
            ),
        );

        expect(
            await screen.findByText(
                'Unable to start Camera 1. Please try again.',
            ),
        ).toBeInTheDocument();

        expect(
            screen.getByText('created'),
        ).toBeInTheDocument();

        expect(fetchMock).toHaveBeenCalledTimes(2);
    });

    it('updates a live stream when the runtime later fails', async () => {
        vi.useFakeTimers();

        try {
            fetchMock
                .mockResolvedValueOnce({
                    ok: true,
                    status: 200,
                    json: async () => [
                        {
                            id: 'stream-1',
                            name: 'Camera 1',
                            url: 'rtsp://localhost:8554/camera-1',
                            state: 'live',
                            createdAt: '2026-10-09T00:00:00Z',
                        },
                    ],
                })
                .mockResolvedValueOnce({
                    ok: true,
                    status: 200,
                    json: async () => ({
                        id: 'stream-1',
                        name: 'Camera 1',
                        url: 'rtsp://localhost:8554/camera-1',
                        state: 'error',
                        error: 'stream source became unavailable',
                        createdAt: '2026-10-09T00:00:00Z',
                    }),
                });

            render(<App />);

            // Flush the initial GET /streams promise and React update.
            await act(async () => {
                await Promise.resolve();
            });

            expect(
                screen.getByText('live'),
            ).toBeInTheDocument();

            expect(
                screen.getByLabelText(
                    /camera 1 player/i,
                ),
            ).toBeInTheDocument();

            // Advance the future live-state reconciliation interval.
            await act(async () => {
                await vi.advanceTimersByTimeAsync(2000);
            });

            expect(
                screen.getByText(
                    'stream source became unavailable',
                ),
            ).toBeInTheDocument();

            expect(
                screen.queryByLabelText(
                    /camera 1 player/i,
                ),
            ).not.toBeInTheDocument();

            expect(fetchMock).toHaveBeenCalledTimes(2);
        } finally {
            vi.useRealTimers();
        }
    });

    it('updates a connecting stream when it becomes live', async () => {
        vi.useFakeTimers();

        try {
            fetchMock
                .mockResolvedValueOnce({
                    ok: true,
                    status: 200,
                    json: async () => [
                        {
                            id: 'stream-1',
                            name: 'Camera 1',
                            url: 'rtsp://localhost:8554/camera-1',
                            state: 'connecting',
                            createdAt: '2026-10-09T00:00:00Z',
                        },
                    ],
                })
                .mockResolvedValueOnce({
                    ok: true,
                    status: 200,
                    json: async () => ({
                        id: 'stream-1',
                        name: 'Camera 1',
                        url: 'rtsp://localhost:8554/camera-1',
                        state: 'live',
                        createdAt: '2026-10-09T00:00:00Z',
                    }),
                });

            render(<App />);

            await act(async () => {
                await Promise.resolve();
            });

            expect(
                screen.getByText('connecting'),
            ).toBeInTheDocument();

            expect(
                screen.queryByLabelText(
                    /camera 1 player/i,
                ),
            ).not.toBeInTheDocument();

            await act(async () => {
                await vi.advanceTimersByTimeAsync(2000);
            });

            expect(
                screen.getByText('live'),
            ).toBeInTheDocument();

            expect(
                screen.getByLabelText(
                    /camera 1 player/i,
                ),
            ).toBeInTheDocument();

            expect(fetchMock).toHaveBeenCalledTimes(2);
        } finally {
            vi.useRealTimers();
        }
    });

    it('reconciles a connecting stream with one request per interval', async () => {
        vi.useFakeTimers();

        try {
            fetchMock
                .mockResolvedValueOnce({
                    ok: true,
                    status: 200,
                    json: async () => [
                        {
                            id: 'stream-1',
                            name: 'Camera 1',
                            url: 'rtsp://localhost:8554/camera-1',
                            state: 'connecting',
                            createdAt: '2026-10-09T00:00:00Z',
                        },
                    ],
                })
                .mockResolvedValueOnce({
                    ok: true,
                    status: 200,
                    json: async () => ({
                        id: 'stream-1',
                        name: 'Camera 1',
                        url: 'rtsp://localhost:8554/camera-1',
                        state: 'connecting',
                        createdAt: '2026-10-09T00:00:00Z',
                    }),
                });

            render(<App />);

            await act(async () => {
                await Promise.resolve();
            });

            expect(
                screen.getByText('connecting'),
            ).toBeInTheDocument();

            expect(fetchMock).toHaveBeenCalledTimes(1);

            await act(async () => {
                await vi.advanceTimersByTimeAsync(2000);
            });

            expect(fetchMock).toHaveBeenCalledTimes(2);

            expect(
                screen.getByText('connecting'),
            ).toBeInTheDocument();
        } finally {
            vi.useRealTimers();
        }
    });

    it('shows an empty state when no streams exist', async () => {
        const user = userEvent.setup();

        render(<App />);

        expect(
            await screen.findByRole('heading', {
                name: /no streams yet/i,
            }),
        ).toBeInTheDocument();

        expect(
            screen.getByText(
                /add an rtsp source to begin monitoring video/i,
            ),
        ).toBeInTheDocument();

        expect(
            screen.getByRole('button', {
                name: /^add stream$/i,
            }),
        ).toBeInTheDocument();

        const emptyStateAction =
            screen.getByRole('button', {
                name: /add first stream/i,
            });

        expect(emptyStateAction).toBeInTheDocument();

        await user.click(emptyStateAction);

        expect(
            screen.getByRole('dialog'),
        ).toBeInTheDocument();
    });
});