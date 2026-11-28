import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { LinkedRunDetails } from './LinkedRunDetails';

const details = vi.hoisted(() => vi.fn());
vi.mock('@inngest/components/RunDetailsV4', () => ({
  RunDetailsV4: (props: { runID: string }) => {
    details(props);
    return <div>Details for {props.runID}</div>;
  },
}));
afterEach(() => {
  cleanup();
  details.mockClear();
});
const getTrigger = vi.fn();

it.each([{ runs: [] }, { runs: [{ id: 'recent-run' }] }])(
  'opens the exact linked run outside the filtered or paginated list: %j',
  ({ runs }) => {
    render(
      <LinkedRunDetails
        runID="older-run"
        runs={runs}
        getTrigger={getTrigger}
        pollInterval={400}
      />,
    );
    expect(screen.getByText('Details for older-run')).toBeTruthy();
    expect(details).toHaveBeenLastCalledWith({
      runID: 'older-run',
      standalone: false,
      getTrigger,
      pollInterval: 400,
    });
  },
);

it('fetches while the list loads, avoids duplicate details when the row arrives, and follows a new link', () => {
  const view = render(
    <LinkedRunDetails runID="target" getTrigger={getTrigger} />,
  );
  expect(screen.getByText('Details for target')).toBeTruthy();
  view.rerender(
    <LinkedRunDetails
      runID="target"
      runs={[{ id: 'target' }]}
      getTrigger={getTrigger}
    />,
  );
  expect(screen.queryByText('Details for target')).toBeNull();
  view.rerender(
    <LinkedRunDetails
      runID="another-run"
      runs={[{ id: 'target' }]}
      getTrigger={getTrigger}
    />,
  );
  expect(screen.getByText('Details for another-run')).toBeTruthy();
  view.rerender(
    <LinkedRunDetails runs={[{ id: 'target' }]} getTrigger={getTrigger} />,
  );
  expect(screen.queryByText(/Details for/)).toBeNull();
});
