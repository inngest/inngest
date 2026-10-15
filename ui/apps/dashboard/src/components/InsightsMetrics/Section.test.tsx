// @vitest-environment jsdom
import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { MAX_INSIGHTS_SQL_BYTES } from '@/components/Insights/insightsSearchParams';
import { Section } from './Section';

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => mocks.navigate,
}));
vi.mock('@inngest/components/DropdownMenu/DropdownMenu', () => ({
  DropdownMenu: ({ children }: React.PropsWithChildren) => children,
  DropdownMenuTrigger: ({ children }: React.PropsWithChildren) => children,
  DropdownMenuContent: ({ children }: React.PropsWithChildren) => children,
  DropdownMenuItem: ({
    children,
    onSelect,
  }: React.PropsWithChildren<{ onSelect: () => void }>) => (
    <button onClick={onSelect}>{children}</button>
  ),
}));
vi.mock('sonner', () => ({ toast: { error: mocks.toastError } }));
vi.mock('@/components/Environments/environment-context', () => ({
  useEnvironment: () => ({ slug: 'production' }),
}));

describe('Insights Metrics section', () => {
  beforeEach(() => {
    mocks.navigate.mockReset();
    mocks.toastError.mockReset();
  });

  it('reports an oversized query instead of silently ignoring the action', () => {
    render(
      <Section
        title="Asymmetric latency"
        query={'x'.repeat(MAX_INSIGHTS_SQL_BYTES + 1)}
      >
        chart
      </Section>,
    );

    fireEvent.click(screen.getByText('Open in Insights'));

    expect(mocks.toastError).toHaveBeenCalledWith(
      'This query is too large to open in Insights.',
    );
    expect(mocks.navigate).not.toHaveBeenCalled();
  });
});
