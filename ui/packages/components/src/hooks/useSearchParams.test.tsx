import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import { act, cleanup, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useBooleanSearchParam } from './useSearchParams';

function BooleanParam() {
  const [value, setValue, remove] = useBooleanSearchParam('excludeDeferred');
  return (
    <>
      <output data-testid="value">{String(value)}</output>
      <button onClick={() => setValue(true)}>Enable</button>
      <button onClick={() => setValue(false)}>Disable</button>
      <button onClick={remove}>Remove</button>
    </>
  );
}

async function setup(search: string) {
  const root = createRootRoute();
  const route = createRoute({ getParentRoute: () => root, path: '/', component: BooleanParam });
  const router = createRouter({
    routeTree: root.addChildren([route]),
    history: createMemoryHistory({ initialEntries: [`/${search}`] }),
  });
  await router.load();
  render(<RouterProvider router={router} />);
  await screen.findByTestId('value');
  return router;
}

beforeEach(() => {
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {});
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe('useBooleanSearchParam', () => {
  it.each([
    ['?excludeDeferred=true', 'true'],
    ['?excludeDeferred=false', 'false'],
    ['?excludeDeferred=%22true%22', 'true'],
    ['?excludeDeferred=%22false%22', 'false'],
    ['', 'undefined'],
    ['?excludeDeferred=other', 'false'],
  ])('reads %s as %s through the router parser', async (search, expected) => {
    await setup(search);
    expect(screen.getByTestId('value').textContent).toBe(expected);
  });

  it('preserves setter/removal behavior and unrelated search parameters', async () => {
    const router = await setup('?other=keep');
    await act(async () => screen.getByText('Enable').click());
    await waitFor(() => expect(screen.getByTestId('value').textContent).toBe('true'));
    await act(async () => screen.getByText('Disable').click());
    await waitFor(() => expect(screen.getByTestId('value').textContent).toBe('false'));
    await act(async () => screen.getByText('Remove').click());
    await waitFor(() => expect(screen.getByTestId('value').textContent).toBe('undefined'));
    expect(router.state.location.search).toEqual({ other: 'keep' });
  });
});
