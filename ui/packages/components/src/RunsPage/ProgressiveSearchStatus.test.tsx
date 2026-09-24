import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { ProgressiveSearchStatus, type ProgressiveSearchPhase } from './ProgressiveSearchStatus';

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe('ProgressiveSearchStatus', () => {
  it('does not report zero matches before the first scan response completes', () => {
    render(
      <ProgressiveSearchStatus phase="searching" hasCompletedScanResponse={false} matchCount={0} />
    );

    expect(screen.getByText('Searching…')).toBeTruthy();
    expect(screen.queryByText(/0 matches/)).toBeNull();
  });

  it('reports zero matches and the relative and absolute frontier after scan progress', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-09-24T12:00:00Z'));
    const frontier = new Date('2026-09-24T10:00:00Z');

    render(
      <ProgressiveSearchStatus
        phase="searching"
        hasCompletedScanResponse
        matchCount={0}
        searchedThrough={frontier}
      />
    );

    expect(screen.getByText(/0 matches so far · Searching/)).toBeTruthy();
    expect(screen.getByText(/2 hours ago/)).toBeTruthy();
    expect(screen.getByText(frontier.toLocaleString(), { exact: false })).toBeTruthy();
  });

  it.each([
    ['searching', '1 match so far · Searching'],
    ['paused', '1 match so far · Automatic search paused'],
    ['complete', '1 match · Search complete'],
    ['cancelled', '1 match so far · Search cancelled'],
    ['error', '1 match so far · Paused after an error'],
  ] satisfies [ProgressiveSearchPhase, string][])('renders the %s result state', (phase, copy) => {
    render(<ProgressiveSearchStatus phase={phase} hasCompletedScanResponse matchCount={1} />);

    expect(screen.getByText(copy)).toBeTruthy();
  });

  it.each([
    ['searching', '0 matches so far'],
    ['complete', '0 matches · Search complete'],
    ['paused', '0 matches so far · Automatic search paused'],
  ] satisfies [ProgressiveSearchPhase, string][])(
    'renders compact %s copy without the frontier',
    (phase, copy) => {
      const frontier = new Date('2026-09-24T10:00:00Z');
      render(
        <ProgressiveSearchStatus
          compact
          phase={phase}
          hasCompletedScanResponse
          matchCount={0}
          searchedThrough={frontier}
        />
      );

      expect(screen.getByText(copy)).toBeTruthy();
      expect(screen.queryByText(frontier.toLocaleString(), { exact: false })).toBeNull();
    }
  );
});
