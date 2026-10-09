import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';

import App from './App';

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
});