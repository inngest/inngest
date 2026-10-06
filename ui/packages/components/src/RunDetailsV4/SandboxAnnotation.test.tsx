/**
 * Sandbox annotations and machine highlight, rendered through the Timeline
 * with the CI-like fixture run.
 */

import type { ReactNode } from 'react';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { Timeline } from './Timeline';
import { sandboxCIRun } from './utils/sandboxTrace.fixture';
import { traceRollup, traceToTimelineData } from './utils/traceConversion';

// Mock modules that use self-referencing @inngest/components/* imports
// which cannot resolve in vitest without a resolve alias.
vi.mock('../Button', () => ({
  Button: () => null,
}));

function Wrapper({ children }: { children: ReactNode }) {
  return <TooltipProvider>{children}</TooltipProvider>;
}

afterEach(() => {
  cleanup();
});

function renderRun(onSelectStep = vi.fn()) {
  const data = traceToTimelineData(traceRollup(sandboxCIRun()), { runID: 'run-1' });
  render(<Timeline data={data} onSelectStep={onSelectStep} />, { wrapper: Wrapper });
  return onSelectStep;
}

function row(name: string): HTMLElement {
  const rows = screen.getAllByTestId('timeline-bar-row');
  const match = rows.find((r) => within(r).queryByText(name, { exact: true }));
  if (!match) throw new Error(`no row ${name}`);
  return match;
}

function chipOf(name: string): HTMLElement {
  return within(row(name)).getByTestId('machine-chip');
}

const isHighlighted = (name: string) =>
  within(row(name)).queryByTestId('dotted-background') !== null;
const isDimmed = (name: string) => row(name).className.includes('opacity-40');

describe('sandbox rows', () => {
  it('renders one annotated row per statement', () => {
    renderRun();

    // Run + its Inngest row + 10 statements; the CI command's 8 steps are one row
    expect(screen.getAllByTestId('timeline-bar-row')).toHaveLength(12);
    expect(screen.queryByText('test › wait #1')).toBeNull();

    const test = within(row('test'));
    expect(test.getByText('pnpm test')).toBeTruthy();
    expect(test.getByTestId('machine-chip').textContent).toBe('ci-build');
    expect(test.getByTestId('exit-badge').textContent).toBe('exit 0');

    expect(within(row('e2e')).getByTestId('exit-badge').textContent).toBe('exit 1');
    expect(chipOf('get-e2e').textContent).toBe('ci-e2e · existing');
    expect(chipOf('e2e-install').textContent).toBe('ci-e2e');
    expect(within(row('notify-start')).queryByTestId('sandbox-annotation')).toBeNull();
  });

  it('pins a machine highlight on chip click without selecting the row', () => {
    const onSelectStep = renderRun();

    fireEvent.click(chipOf('install'));

    expect(onSelectStep).not.toHaveBeenCalled();
    expect(chipOf('install').getAttribute('aria-pressed')).toBe('true');
    for (const name of ['build-machine', 'install', 'test', 'snapshot-build', 'destroy-build']) {
      expect(isHighlighted(name)).toBe(true);
      expect(isDimmed(name)).toBe(false);
    }
    for (const name of ['get-e2e', 'e2e', 'notify-start']) {
      expect(isHighlighted(name)).toBe(false);
      expect(isDimmed(name)).toBe(true);
    }
    expect(isDimmed('Run')).toBe(false);

    // Clicking again clears it
    fireEvent.click(chipOf('test'));
    expect(isHighlighted('install')).toBe(false);
    expect(isDimmed('e2e')).toBe(false);
  });

  it('clears a pinned highlight on Escape', () => {
    renderRun();

    fireEvent.click(chipOf('e2e'));
    expect(isHighlighted('e2e-release')).toBe(true);

    fireEvent.keyDown(window, { key: 'Escape' });
    expect(isHighlighted('e2e-release')).toBe(false);
  });

  it('previews on hover only while nothing is pinned', () => {
    renderRun();

    fireEvent.mouseEnter(chipOf('e2e'));
    expect(isHighlighted('e2e-install')).toBe(true);
    expect(isDimmed('install')).toBe(true);
    fireEvent.mouseLeave(chipOf('e2e'));
    expect(isHighlighted('e2e-install')).toBe(false);

    fireEvent.click(chipOf('install'));
    fireEvent.mouseEnter(chipOf('e2e'));
    expect(isHighlighted('install')).toBe(true);
    expect(isHighlighted('e2e-install')).toBe(false);
  });

  it('selects the statement row when the row itself is clicked', () => {
    const onSelectStep = renderRun();

    fireEvent.click(row('test'));
    expect(onSelectStep).toHaveBeenCalledWith(expect.stringMatching(/-statement$/));
  });
});
