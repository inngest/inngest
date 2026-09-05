import { cleanup, render, screen } from '@testing-library/react';
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest';

import { Pill } from './Pill';

class ResizeObserverMock {
  disconnect() {}
  observe() {}
  unobserve() {}
}

beforeAll(() => vi.stubGlobal('ResizeObserver', ResizeObserverMock));
afterAll(() => vi.unstubAllGlobals());
afterEach(cleanup);

describe('Pill', () => {
  it('renders a trailing action separately from truncatable content', () => {
    render(<Pill action={<a href="/billing/plans">Upgrade</a>}>Execution limit reached</Pill>);

    expect(screen.getByText('Execution limit reached')).toBeTruthy();

    const action = screen.getByRole('link', { name: 'Upgrade' });
    expect(action.getAttribute('href')).toBe('/billing/plans');
    expect(action.parentElement?.className).toContain('shrink-0');
  });
});
