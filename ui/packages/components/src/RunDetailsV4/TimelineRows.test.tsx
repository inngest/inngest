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

const renderFixture = () => {
  return render(
    <Timeline data={traceToTimelineData(traceRollup(stepSpansTrace), { runID: 'run' })} />,
    { wrapper: Wrapper }
  );
};

const rowFor = (rowName: string) => {
  return screen.getByText(rowName).closest('[data-testid=timeline-bar-row]') as HTMLElement;
};

const badgeFor = (rowName: string) => {
  return rowFor(rowName).querySelector<HTMLElement>('[data-testid=sandbox-badge]');
};

const kindTagsFor = (rowName: string) => {
  return [...rowFor(rowName).querySelectorAll('[data-testid=group-kind-tag]')].map((tag) => {
    return tag.textContent;
  });
};

describe('group kind tags', () => {
  it('tags a span group with its kind, and nothing else', () => {
    renderFixture();

    expect(kindTagsFor('base')).toEqual(['job']);
    expect(kindTagsFor('e2e')).toEqual(['job']);
    expect(kindTagsFor('Research network')).toEqual([]);
    expect(kindTagsFor('notify')).toEqual([]);

    fireEvent.click(screen.getByText('Research network'));
    expect(kindTagsFor('Research agent')).toEqual(['agent']);
  });
});

describe('sandbox badge', () => {
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
});
