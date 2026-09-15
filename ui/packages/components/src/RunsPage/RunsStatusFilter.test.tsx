import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';

import RunsStatusFilter from './RunsStatusFilter';

vi.mock('../Filter/StatusFilter', () => ({
  default: ({ availableStatuses }: { availableStatuses: string[] }) => (
    <div>
      {availableStatuses.map((status) => (
        <span key={status}>{status}</span>
      ))}
    </div>
  ),
}));

afterEach(cleanup);

it('offers supported statuses but not the UNKNOWN output fallback', () => {
  render(<RunsStatusFilter selectedStatuses={[]} onStatusesChange={() => undefined} />);

  expect(screen.getByText('RUNNING')).toBeTruthy();
  expect(screen.getByText('COMPLETED')).toBeTruthy();
  expect(screen.queryByText('UNKNOWN')).toBeNull();
});
