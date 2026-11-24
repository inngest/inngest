import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import { parseDevServerSearch } from '@/search-params';

import { Route } from './route';

vi.mock('@/store/generated', () => ({
  AppMethod: { Api: 'API' },
  FunctionTriggerTypes: { Event: 'EVENT' },
  useGetFunctionsQuery: () => ({ data: { functions: [] }, isFetching: false }),
  useInvokeFunctionMutation: () => [vi.fn()],
}));

beforeEach(() => {
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {});
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

test.each(['orders', '123', 'false', '1e3', '9007199254740993'])(
  'keeps the function filter from a direct URL: %s',
  async (query) => {
    const root = createRootRoute();
    const functions = createRoute({
      getParentRoute: () => root,
      path: '/functions',
      component: Route.options.component,
    });
    const router = createRouter({
      routeTree: root.addChildren([functions]),
      history: createMemoryHistory({
        initialEntries: [`/functions?query=${query}`],
      }),
      parseSearch: parseDevServerSearch,
    });
    await router.load();
    render(<RouterProvider router={router} />);
    const input = await screen.findByPlaceholderText<HTMLInputElement>(
      'Search by function name',
    );
    expect(input.value).toBe(query);
  },
);

test('preserves the router defaults for other keys and structured values', () => {
  expect(parseDevServerSearch('?query=%22123%22&other=true&count=4')).toEqual({
    query: '123',
    other: true,
    count: 4,
  });
  expect(parseDevServerSearch('?query=%5B1%2C2%5D')).toEqual({ query: [1, 2] });
});
