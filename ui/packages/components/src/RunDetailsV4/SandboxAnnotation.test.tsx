/**
 * Sandbox annotations and machine highlight, rendered through the Timeline
 * with the CI-like fixture run.
 */

import type { ReactNode } from 'react';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { machineColor, shortMachineLabel } from './SandboxAnnotation';
import { Timeline, collectMachineIds } from './Timeline';
import {
  BUILD_MACHINE_NAME,
  BUILD_SANDBOX_ID,
  E2E_SANDBOX_ID,
  TEST_NAME,
  ciTestSteps,
  sandboxCIRun,
} from './utils/sandboxTrace.fixture';
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

    // Run + its Inngest row + 12 statements; the CI command's 8 steps are one row
    expect(screen.getAllByTestId('timeline-bar-row')).toHaveLength(14);
    expect(screen.queryByText(`${TEST_NAME} › wait #1`)).toBeNull();
    expect(screen.queryByText('build › machine › setup')).toBeNull();

    const test = within(row(TEST_NAME));
    // The title already ends with the command: just the verb and the machine
    expect(test.getByTestId('sandbox-annotation').textContent).toBe('Runonci-01M…-buildexit 0');
    expect(test.getByTestId('machine-chip').textContent).toBe('ci-01M…-build');
    expect(test.getByTestId('machine-chip').getAttribute('title')).toContain(BUILD_MACHINE_NAME);
    expect(test.getByTestId('exit-badge').textContent).toBe('exit 0');

    // A title without the command still shows it
    expect(within(row('install')).getByText('pnpm install')).toBeTruthy();

    expect(within(row('e2e')).getByTestId('exit-badge').textContent).toBe('exit 1');
    expect(chipOf('get-e2e').textContent).toBe('ci-01M…-e2e · existing');
    expect(chipOf('e2e-install').textContent).toBe('ci-01M…-e2e');
    expect(within(row('build › pause')).getByText('Pause')).toBeTruthy();
    expect(within(row('notify-start')).queryByTestId('sandbox-annotation')).toBeNull();
  });

  it('pins a machine highlight on chip click without selecting the row', () => {
    const onSelectStep = renderRun();

    fireEvent.click(chipOf('install'));

    expect(onSelectStep).not.toHaveBeenCalled();
    expect(chipOf('install').getAttribute('aria-pressed')).toBe('true');
    for (const name of [
      'build › machine',
      'install',
      TEST_NAME,
      'snapshot-build',
      'destroy-build',
    ]) {
      expect(isHighlighted(name)).toBe(true);
      expect(isDimmed(name)).toBe(false);
    }
    for (const name of ['get-e2e', 'e2e', 'notify-start']) {
      expect(isHighlighted(name)).toBe(false);
      expect(isDimmed(name)).toBe(true);
    }
    expect(isDimmed('Run')).toBe(false);

    // Clicking again clears it
    fireEvent.click(chipOf(TEST_NAME));
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

    fireEvent.click(row(TEST_NAME));
    expect(onSelectStep).toHaveBeenCalledWith(expect.stringMatching(/-statement$/));
  });
});

describe('machine chips', () => {
  it('shortens a CI machine name in the middle, keeping the job', () => {
    expect(shortMachineLabel('ci-01M48D0A2NS1T6C051P07DC8RB-base')).toBe('ci-01M…-base');
    expect(shortMachineLabel('ci-01M48D0A2NS1T6C051P07DC8RB-test-probe')).toBe(
      'ci-01M…-test-probe'
    );
    expect(shortMachineLabel('short-name')).toBe('short-name');
    expect(shortMachineLabel('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa')).toBe('aaaaaa…aaaaaaaaaaaaa');
  });

  it('colours machines by first appearance, so neighbours differ', () => {
    const data = traceToTimelineData(traceRollup(sandboxCIRun()), { runID: 'run-1' });
    const ids = collectMachineIds(data.bars);

    expect(ids).toEqual([BUILD_SANDBOX_ID, E2E_SANDBOX_ID]);
    expect(machineColor(ids[0]!, ids)).not.toBe(machineColor(ids[1]!, ids));

    const three = ['a', 'b', 'c'];
    const colours = three.map((id) => machineColor(id, three));
    expect(new Set(colours).size).toBe(3);
  });
});

describe('retried commands', () => {
  it('draws the failed attempt red and counts attempts in the badge', () => {
    const steps = [
      ...ciTestSteps({ attempt: 1, exitCode: 1 }),
      ...ciTestSteps({ attempt: 2, offset: 100 }),
    ];
    const root = sandboxCIRun();
    const data = traceToTimelineData(traceRollup({ ...root, childrenSpans: steps }), {
      runID: 'run-1',
    });
    render(<Timeline data={data} onSelectStep={vi.fn()} />, { wrapper: Wrapper });

    const retried = row(TEST_NAME);
    expect(within(retried).getByTestId('exit-badge').textContent).toBe('exit 0 · 2 attempts');

    const segments = [...retried.querySelectorAll('[data-segment-style]')];
    expect(segments.map((s) => s.getAttribute('data-segment-style'))).toEqual([
      'sandbox.active',
      'sandbox.waiting',
      'sandbox.active',
      'sandbox.active',
      'sandbox.waiting',
      'sandbox.active',
    ]);
    // Attempt 1 is in the failed colour; the waiting state is a hollow outline
    expect(segments[0]!.className).toContain('bg-status-failed');
    expect(segments[1]!.className).toContain('border-status-failed');
    expect(segments[4]!.className).toContain('border-status-completed');
    // One continuous fill per state, never the tick pattern
    for (const segment of segments) {
      expect((segment as HTMLElement).style.maskImage).toBe('');
    }
  });
});
