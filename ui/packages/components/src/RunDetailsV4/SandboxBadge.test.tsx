import type { ReactNode } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { numberSandboxes, sandboxBarData } from './SandboxBadge';
import { Timeline } from './Timeline';
import type { TimelineBarData } from './TimelineBar.types';
import { traceWalk } from './runDetailsUtils';
import type { Trace } from './types';
import { SANDBOX_A, SANDBOX_B, SANDBOX_C, stepSpansTrace } from './utils/stepSpans.fixture';
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

const span = (name: string): Trace => {
  let found: Trace | undefined;

  traceWalk(stepSpansTrace, (t) => {
    if (!found && t.name === name) found = t;
  });

  return found!;
};

const groupOf = (...children: Trace[]): Trace => {
  return { ...span('base'), childrenSpans: children };
};

const fixtureBars = () => {
  return traceToTimelineData(traceRollup(stepSpansTrace), { runID: 'run' }).bars;
};

describe('sandboxBarData', () => {
  it('gives a step its own sandbox, named for the tooltip', () => {
    expect(sandboxBarData(span('Create sandbox'))).toEqual({
      sandboxId: SANDBOX_A.sandbox_id,
      label: 'ci-01JB7Q2XKZ-base',
    });
  });

  it('gives no sandbox to steps without a sandbox ID', () => {
    expect(sandboxBarData(span('notify'))).toBeUndefined();
    expect(sandboxBarData(span('Wait for snapshot'))).toBeUndefined();
  });

  it('gives a group the sandbox most of its direct children use', () => {
    // Start sandbox, `$ pnpm e2e` and Save sandbox on B outvote `api` on C
    expect(sandboxBarData(span('e2e'))?.sandboxId).toBe(SANDBOX_B.sandbox_id);
  });

  it('counts a subgroup as one vote for its own sandbox', () => {
    expect(sandboxBarData(span('$ pnpm test'))?.sandboxId).toBe(SANDBOX_A.sandbox_id);

    const oneSubgroupOnA = groupOf(span('$ pnpm test'), span('api'), span('$ pnpm e2e'));
    expect(sandboxBarData(oneSubgroupOnA)).toBeUndefined();
  });

  it("doesn't count children without a sandbox", () => {
    expect(sandboxBarData(span('Snapshot sandbox'))?.sandboxId).toBe(SANDBOX_A.sandbox_id);
    expect(sandboxBarData(span('Research network'))).toBeUndefined();
  });

  it('gives a tied group no sandbox', () => {
    expect(sandboxBarData(groupOf(span('$ pnpm lint'), span('$ pnpm e2e')))).toBeUndefined();
  });
});

describe('numberSandboxes', () => {
  it('numbers sandboxes by when a step first uses them', () => {
    expect([...numberSandboxes(fixtureBars())]).toEqual([
      [SANDBOX_A.sandbox_id, 1],
      [SANDBOX_B.sandbox_id, 2],
      [SANDBOX_C.sandbox_id, 3],
    ]);
  });

  it('follows run time, not row order', () => {
    const bar = (id: string, secs: number): TimelineBarData => {
      return {
        id,
        name: id,
        startTime: new Date(secs * 1000),
        endTime: null,
        style: 'step.run',
        sandbox: { sandboxId: id, label: id },
      };
    };

    expect([...numberSandboxes([bar('late', 9), bar('early', 1)])]).toEqual([
      ['early', 1],
      ['late', 2],
    ]);
  });
});

describe('sandbox badge', () => {
  const renderFixture = () => {
    return render(
      <Timeline data={traceToTimelineData(traceRollup(stepSpansTrace), { runID: 'run' })} />,
      {
        wrapper: Wrapper,
      }
    );
  };

  const rowFor = (rowName: string) => {
    return screen.getByText(rowName).closest('[data-testid=timeline-bar-row]') as HTMLElement;
  };

  const badgeFor = (rowName: string) => {
    return rowFor(rowName).querySelector<HTMLElement>('[data-testid=sandbox-badge]');
  };

  it('shows S1, S2 on top-level rows, after the kind tag', () => {
    renderFixture();

    expect(badgeFor('base')?.textContent).toBe('S1');
    expect(badgeFor('base')?.title).toBe('ci-01JB7Q2XKZ-base');
    expect(badgeFor('e2e')?.textContent).toBe('S2');
    expect(badgeFor('notify')).toBeNull();
    expect(badgeFor('GitHub')).toBeNull();
    expect(badgeFor('Clean up sandboxes')).toBeNull();
    expect(badgeFor('Research network')).toBeNull();

    const name = screen.getByText('e2e');
    expect([...name.children].map((el) => el.getAttribute('data-testid'))).toEqual([
      'group-kind-tag',
      'sandbox-badge',
    ]);
  });

  it('hides a badge matching the row above and shows one that differs', () => {
    renderFixture();

    fireEvent.click(screen.getByText('base'));
    expect(badgeFor('$ pnpm lint')).toBeNull();
    expect(badgeFor('$ pnpm test')).toBeNull();

    fireEvent.click(screen.getByText('e2e'));
    expect(badgeFor('$ pnpm e2e')).toBeNull();
    expect(badgeFor('api')?.textContent).toBe('S3');
    expect(badgeFor('api')?.title).toBe('ci-01JB7Q2XKZ-api');

    fireEvent.click(screen.getByText('api'));
    expect(badgeFor('$ pnpm api')).toBeNull();
  });

  it('keeps rows single-line with no machine tag or highlight', () => {
    renderFixture();
    fireEvent.click(screen.getByText('base'));
    fireEvent.click(screen.getByText('e2e'));

    expect(screen.queryByTestId('machine-tag')).toBeNull();
    expect(screen.queryByTestId('machine-highlight')).toBeNull();
    expect(screen.queryByTestId('sandbox-annotation')).toBeNull();
    expect(screen.queryByText(/pnpm install --frozen-lockfile on/)).toBeNull();

    for (const row of screen.getAllByTestId('timeline-bar-row')) {
      expect(row.style.height).toBe('28px');
    }
  });
});
