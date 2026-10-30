import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';

import RunsTable from './RunsTable';
import type { Run } from './types';

vi.mock('./columns', () => ({
  useScopedColumns: () => [{ id: 'id', header: 'Run', accessorKey: 'id' }],
}));
afterEach(cleanup);
const run = (id: string) => ({ id } as Run);
const props = {
  scope: 'env' as const,
  getRowCanExpand: () => true,
  renderSubComponent: (value: Run) => <div>Details for {value.id}</div>,
};

it('opens the linked run when its data arrives and preserves it when polling prepends another run', () => {
  const view = render(<RunsTable {...props} data={[]} initialExpandedRunID="target" />);
  view.rerender(<RunsTable {...props} data={[run('target')]} initialExpandedRunID="target" />);
  expect(screen.getByText('Details for target')).toBeTruthy();
  view.rerender(
    <RunsTable {...props} data={[run('newer'), run('target')]} initialExpandedRunID="target" />
  );
  expect(screen.getByText('Details for target')).toBeTruthy();
  expect(screen.queryByText('Details for newer')).toBeNull();
  fireEvent.click(screen.getByText('target'));
  view.rerender(
    <RunsTable {...props} data={[run('newer'), run('target')]} initialExpandedRunID="target" />
  );
  expect(screen.queryByText('Details for target')).toBeNull();
});

it('opens a different linked run on navigation without changing ordinary manual expansion', () => {
  const data = [run('first'), run('second')];
  const view = render(<RunsTable {...props} data={data} />);
  expect(screen.queryByText('Details for first')).toBeNull();
  fireEvent.click(screen.getByText('first'));
  expect(screen.getByText('Details for first')).toBeTruthy();
  view.rerender(<RunsTable {...props} data={data} initialExpandedRunID="second" />);
  expect(screen.getByText('Details for second')).toBeTruthy();
});
