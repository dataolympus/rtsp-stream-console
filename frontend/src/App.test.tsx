import { render, screen } from '@testing-library/react';
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
});