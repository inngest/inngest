import type { ReactNode } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { Timeline } from './Timeline';
import { stepSpansTrace } from './utils/stepSpans.fixture';
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

/** The kind tags on a row's name line, leaving out its MACHINE tag */
const kindTagsFor = (rowName: string) =>
  [...screen.getByText(rowName).querySelectorAll('[data-testid=span-kind-tag]')].map(
    (tag) => tag.textContent
  );

describe('span kind tags', () => {
  it('tags a span group with its kind, upper-cased, before its name', () => {
    render(<Timeline data={traceToTimelineData(traceRollup(stepSpansTrace), { runID: 'run' })} />, {
      wrapper: Wrapper,
    });

    expect(kindTagsFor('lint')).toEqual(['CMD']);
    expect(kindTagsFor('e2e')).toEqual(['JOB']);
    expect(kindTagsFor('Research network')).toEqual([]);
    expect(kindTagsFor('notify')).toEqual([]);

    fireEvent.click(screen.getByText('Research network'));
    expect(kindTagsFor('Research agent')).toEqual(['AGENT']);
  });
});
