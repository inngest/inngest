import { describe, expect, it } from 'vitest';

import type { TimelineBarData } from '../TimelineBar.types';
import { traceWalk } from '../runDetailsUtils';
import type { Trace } from '../types';
import { badgeSandboxes, sandboxBarData } from './sandboxes';
import { SANDBOX_A, SANDBOX_B, stepSpansTrace } from './stepSpans.fixture';

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

describe('sandboxBarData', () => {
  it('gives a step its own sandbox, named for the tooltip', () => {
    expect(sandboxBarData(span('Create sandbox'))).toEqual({
      id: SANDBOX_A.sandbox_id,
      label: 'ci-01JB7Q2XKZ-base',
    });
  });

  it('gives no sandbox to steps without a sandbox ID', () => {
    expect(sandboxBarData(span('notify'))).toBeUndefined();
    expect(sandboxBarData(span('Wait for snapshot'))).toBeUndefined();
  });

  it('gives a group the sandbox most of its direct children use', () => {
    // Start sandbox, `$ pnpm e2e` and Save sandbox on B outvote `api` on C
    expect(sandboxBarData(span('e2e'))?.id).toBe(SANDBOX_B.sandbox_id);
  });

  it('counts a subgroup as one vote for its own sandbox', () => {
    expect(sandboxBarData(span('$ pnpm test'))?.id).toBe(SANDBOX_A.sandbox_id);

    const oneSubgroupOnA = groupOf(span('$ pnpm test'), span('api'), span('$ pnpm e2e'));
    expect(sandboxBarData(oneSubgroupOnA)).toBeUndefined();
  });

  it("doesn't count children without a sandbox", () => {
    expect(sandboxBarData(span('Snapshot sandbox'))?.id).toBe(SANDBOX_A.sandbox_id);
    expect(sandboxBarData(span('Research network'))).toBeUndefined();
  });

  it('gives a tied group no sandbox', () => {
    expect(sandboxBarData(groupOf(span('$ pnpm lint'), span('$ pnpm e2e')))).toBeUndefined();
  });
});

describe('badgeSandboxes', () => {
  const bar = (
    id: string,
    secs: number,
    children?: TimelineBarData[],
    sandboxID = id
  ): TimelineBarData => {
    return {
      id,
      name: id,
      startTime: new Date(secs * 1000),
      endTime: null,
      style: 'step.run',
      sandbox: { id: sandboxID, label: sandboxID },
      children,
    };
  };

  const badges = (bars: TimelineBarData[]): Record<string, string | undefined> => {
    badgeSandboxes(bars);

    const out: Record<string, string | undefined> = {};
    const walk = (list: TimelineBarData[]) => {
      for (const b of list) {
        out[b.id] = b.sandbox?.badge;
        walk(b.children ?? []);
      }
    };
    walk(bars);

    return out;
  };

  it('orders siblings by start time, not row order', () => {
    expect(badges([bar('late', 9), bar('early', 1)])).toEqual({ early: 'S1', late: 'S2' });
  });

  it("numbers a group's own sandbox before an extra one inside it that starts first", () => {
    const job = bar('group', 5, [bar('extra', 1), bar('step', 6, undefined, 'job')], 'job');

    expect(badges([bar('base', 0), job])).toEqual({
      base: 'S1',
      group: 'S2',
      extra: 'S3',
      step: undefined,
    });
  });
});
