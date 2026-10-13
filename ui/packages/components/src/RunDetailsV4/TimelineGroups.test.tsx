/**
 * Span groups start collapsed; the path to a failed or running step starts open.
 */

import type { ReactNode } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { Timeline } from './Timeline';
import type { TimelineBarData, TimelineData } from './TimelineBar.types';
import { collectAutoExpandedGroupIds } from './utils/groupExpansion';

vi.mock('../Button', () => ({
  Button: ({
    onClick,
    ...props
  }: {
    onClick?: () => void;
    'aria-label'?: string;
    icon?: ReactNode;
  }) => (
    <button onClick={onClick} aria-label={props['aria-label']}>
      {props.icon}
    </button>
  ),
}));

function Wrapper({ children }: { children: ReactNode }) {
  return <TooltipProvider>{children}</TooltipProvider>;
}

afterEach(() => {
  cleanup();
});

const start = new Date('2024-01-01T00:00:00Z');
const end = new Date('2024-01-01T00:00:10Z');

function step(id: string, status: string, children?: TimelineBarData[]): TimelineBarData {
  return {
    id,
    name: id,
    startTime: start,
    endTime: status === 'RUNNING' ? null : end,
    style: 'step.run',
    status,
    children,
  };
}

function group(id: string, children: TimelineBarData[], status = 'COMPLETED'): TimelineBarData {
  return {
    id,
    name: id,
    startTime: start,
    endTime: end,
    style: 'span.group',
    status,
    children,
  };
}

function timeline(...children: TimelineBarData[]): TimelineData {
  return {
    minTime: start,
    maxTime: end,
    leftWidth: 40,
    bars: [
      {
        id: 'root',
        name: 'Run',
        startTime: start,
        endTime: end,
        style: 'root',
        isRoot: true,
        status: 'COMPLETED',
        children,
      },
    ],
  };
}

describe('collectAutoExpandedGroupIds', () => {
  it('returns every group on the path to a failed or running step, and no other row', () => {
    const bars = timeline(
      group('quiet', [step('ok', 'COMPLETED')]),
      group('outer', [group('inner', [step('bad', 'FAILED')])]),
      group('live', [step('going', 'RUNNING')]),
      step('plain', 'COMPLETED', [step('plain-bad', 'FAILED')])
    ).bars;

    expect([...collectAutoExpandedGroupIds(bars)].sort()).toEqual(['inner', 'live', 'outer']);
  });
});

describe('Timeline span groups', () => {
  it('starts groups collapsed, showing only the group row', () => {
    render(<Timeline data={timeline(group('g', [step('a', 'COMPLETED')]))} />, {
      wrapper: Wrapper,
    });

    expect(screen.getByText('g')).toBeTruthy();
    expect(screen.queryByText('a')).toBeNull();
  });

  it('starts the ancestors of a failed step expanded', () => {
    render(
      <Timeline data={timeline(group('outer', [group('inner', [step('bad', 'FAILED')])]))} />,
      { wrapper: Wrapper }
    );

    expect(screen.getByText('bad')).toBeTruthy();
  });

  it('starts the ancestors of a running step expanded, and leaves other groups collapsed', () => {
    render(
      <Timeline
        data={timeline(
          group('live', [step('going', 'RUNNING')]),
          group('done', [step('finished', 'COMPLETED')])
        )}
      />,
      { wrapper: Wrapper }
    );

    expect(screen.getByText('going')).toBeTruthy();
    expect(screen.queryByText('finished')).toBeNull();
  });

  it('keeps a group the user opened open when the data updates', () => {
    const data = timeline(group('quiet', [step('a', 'COMPLETED')]));
    const { rerender } = render(<Timeline data={data} />, { wrapper: Wrapper });

    fireEvent.click(screen.getByText('quiet'));
    expect(screen.getByText('a')).toBeTruthy();

    rerender(<Timeline data={timeline(group('quiet', [step('a', 'COMPLETED')]))} />);
    expect(screen.getByText('a')).toBeTruthy();
  });

  it('keeps an auto-expanded group the user collapsed collapsed when the data updates', () => {
    const { rerender } = render(
      <Timeline data={timeline(group('loud', [step('bad', 'FAILED')]))} />,
      { wrapper: Wrapper }
    );
    expect(screen.getByText('bad')).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: 'Collapse' }));
    expect(screen.queryByText('bad')).toBeNull();

    rerender(<Timeline data={timeline(group('loud', [step('bad', 'FAILED')]))} />);
    expect(screen.queryByText('bad')).toBeNull();
  });

  it('does not auto-expand a plain step with children', () => {
    render(<Timeline data={timeline(step('plain', 'COMPLETED', [step('child', 'FAILED')]))} />, {
      wrapper: Wrapper,
    });

    expect(screen.queryByText('child')).toBeNull();
  });

  it('collapse all closes auto-expanded groups and expand all opens collapsed ones', () => {
    render(
      <Timeline
        data={timeline(
          group('quiet', [step('a', 'COMPLETED')]),
          group('loud', [step('bad', 'FAILED')])
        )}
      />,
      { wrapper: Wrapper }
    );

    fireEvent.click(screen.getByRole('button', { name: /collapse all/i }));
    expect(screen.queryByText('bad')).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: /expand all/i }));
    expect(screen.getByText('a')).toBeTruthy();
    expect(screen.getByText('bad')).toBeTruthy();
  });
});
