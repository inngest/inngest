import type { ReactNode } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { Timeline } from './Timeline';
import { traceWalk } from './runDetailsUtils';
import { CI_ORIGIN, SDK_ORIGIN, stepSpansTrace } from './utils/stepSpans.fixture';
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

/** Every kind tag on a row */
const kindTagsFor = (rowName: string) => {
  const row = screen.getByText(rowName).closest('[data-testid=timeline-bar-row]')!;

  return [...row.querySelectorAll('[data-testid=group-kind-tag]')].map((tag) => {
    return tag.textContent;
  });
};

describe('group kind tags', () => {
  it('tags a span group with its kind, upper-cased, before its name', () => {
    render(<Timeline data={traceToTimelineData(traceRollup(stepSpansTrace), { runID: 'run' })} />, {
      wrapper: Wrapper,
    });

    expect(kindTagsFor('base')).toEqual(['JOB']);
    expect(kindTagsFor('e2e')).toEqual(['JOB']);
    expect(kindTagsFor('Research network')).toEqual([]);
    expect(kindTagsFor('notify')).toEqual([]);
    expect(kindTagsFor('GitHub')).toEqual([]);

    fireEvent.click(screen.getByText('Research network'));
    expect(kindTagsFor('Research agent')).toEqual(['AGENT']);
  });

  it('shows at most one tag per row', () => {
    render(<Timeline data={traceToTimelineData(traceRollup(stepSpansTrace), { runID: 'run' })} />, {
      wrapper: Wrapper,
    });

    fireEvent.click(screen.getByText('base'));
    fireEvent.click(screen.getByText('e2e'));
    fireEvent.click(screen.getByText('Research network'));
    fireEvent.click(screen.getByText('Research agent'));

    for (const row of screen.getAllByTestId('timeline-bar-row')) {
      expect(row.querySelectorAll('[data-testid=group-kind-tag]').length).toBeLessThanOrEqual(1);
    }

    expect(kindTagsFor('$ pnpm lint')).toEqual([]);
    expect(kindTagsFor('api')).toEqual([]);
    expect(kindTagsFor('search tool')).toEqual(['TOOL']);
  });
});

describe('fixture origins', () => {
  it('marks only what CI does itself, never jobs, commands or `api`', () => {
    const origins = new Map<string, string | null | undefined>();

    traceWalk(stepSpansTrace, (t) => {
      origins.set(t.name, t.origin);
    });

    for (const name of ['base', 'e2e', 'api', '$ pnpm test', '$ pnpm dev', 'notify']) {
      expect(origins.get(name) ?? null).toBeNull();
    }

    for (const name of ['GitHub', 'Start sandbox', 'Attempt 1', 'Save sandbox', 'Read output']) {
      expect(origins.get(name)).toBe(CI_ORIGIN);
    }

    expect(origins.get('Create snapshot')).toBe(SDK_ORIGIN);
  });
});
