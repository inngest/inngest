import type { ReactNode } from 'react';
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { sandboxBarData, shortMachineLabel } from './SandboxAnnotation';
import { Timeline } from './Timeline';
import { traceWalk } from './runDetailsUtils';
import type { Trace } from './types';
import { MACHINE_A, stepSpansTrace } from './utils/stepSpans.fixture';
import { traceRollup, traceToTimelineData } from './utils/traceConversion';

vi.mock('../Button', () => ({
  Button: () => null,
}));

function Wrapper({ children }: { children: ReactNode }) {
  return <TooltipProvider>{children}</TooltipProvider>;
}

afterEach(() => {
  cleanup();
});

const child = (name: string): Trace => {
  let found: Trace | undefined;
  traceWalk(stepSpansTrace, (t) => {
    if (!found && t.name === name) found = t;
  });
  return found!;
};

describe('sandboxBarData', () => {
  it('describes a sandbox step', () => {
    expect(sandboxBarData(child('setup'))).toEqual({
      sandboxId: MACHINE_A.sandbox_id,
      machineLabel: 'ci-01JB7Q2XKZ-base',
      command: 'pnpm install --frozen-lockfile',
    });
  });

  it('ignores steps without sandbox metadata', () => {
    expect(sandboxBarData(child('notify'))).toBeUndefined();
    expect(sandboxBarData(child('Research network'))).toBeUndefined();
  });

  it('gives a group its shared machine', () => {
    expect(sandboxBarData(child('test'))).toEqual({
      sandboxId: MACHINE_A.sandbox_id,
      machineLabel: 'ci-01JB7Q2XKZ-base',
    });
  });

  it("doesn't let steps that name no machine hide a group's machine", () => {
    expect(sandboxBarData(child('snapshot'))?.sandboxId).toBe(MACHINE_A.sandbox_id);
  });

  it('shows no machine for a group spanning two machines', () => {
    const mixed: Trace = {
      ...child('lint'),
      childrenSpans: [...child('machine').childrenSpans!, ...child('e2e').childrenSpans!],
    };
    expect(sandboxBarData(mixed)).toBeUndefined();
  });
});

describe('shortMachineLabel', () => {
  it('keeps short labels and shortens long ones in the middle', () => {
    expect(shortMachineLabel('box')).toBe('box');
    expect(shortMachineLabel('ci-01JB7Q2XKZABCDEF-base')).toBe('ci-01J…KZABCDEF-base');
  });
});

describe('machine highlight', () => {
  const renderFixture = () =>
    render(<Timeline data={traceToTimelineData(traceRollup(stepSpansTrace), { runID: 'run' })} />, {
      wrapper: Wrapper,
    });

  const rowFor = (rowName: string) =>
    screen.getByText(rowName).closest('[data-testid=timeline-bar-row]') as HTMLElement;

  const machineTagFor = (rowName: string) =>
    rowFor(rowName).querySelector('[data-testid=machine-tag]') as HTMLElement;

  it('shows a MACHINE tag on sandbox rows and groups', () => {
    renderFixture();

    // machine, lint, dev server, test, snapshot, e2e
    expect(screen.getAllByTestId('machine-tag')).toHaveLength(6);

    const tag = machineTagFor('e2e');
    expect(tag.textContent).toBe('MACHINEci-01JB7Q2XKZ-e2e');
    expect(tag.title).toMatch(/^ci-01JB7Q2XKZ-e2e\n/);
  });

  it('pins a highlight on every row of a machine until Escape', () => {
    renderFixture();

    fireEvent.click(machineTagFor('e2e'));
    expect(machineTagFor('e2e').getAttribute('aria-pressed')).toBe('true');
    expect(screen.getAllByTestId('machine-highlight')).toHaveLength(1);

    // Rows inside a highlighted group are highlighted too
    fireEvent.click(screen.getByText('e2e'));
    expect(screen.getAllByTestId('machine-highlight')).toHaveLength(3);

    act(() => {
      fireEvent.keyDown(window, { key: 'Escape' });
    });
    expect(screen.queryByTestId('machine-highlight')).toBeNull();
  });

  it('previews a machine on hover', () => {
    renderFixture();

    fireEvent.mouseEnter(machineTagFor('machine'));
    expect(screen.getAllByTestId('machine-highlight')).toHaveLength(5);

    fireEvent.mouseLeave(machineTagFor('machine'));
    expect(screen.queryByTestId('machine-highlight')).toBeNull();
  });
});
