import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useRunCount } from './useRunCount';

const mocks = vi.hoisted(() => ({
  useSkippableGraphQLQuery: vi.fn(),
}));

vi.mock('@/gql', () => ({
  graphql: vi.fn(() => ({})),
}));

vi.mock('@/utils/useGraphQLQuery', () => ({
  useSkippableGraphQLQuery: mocks.useSkippableGraphQLQuery,
}));

describe('useRunCount', () => {
  beforeEach(() => {
    mocks.useSkippableGraphQLQuery.mockReset();
  });

  it('preserves a GraphQL count error when null bubbling removes the function', () => {
    const error = new Error('error counting active runs');
    mocks.useSkippableGraphQLQuery.mockReturnValue({
      data: { environment: { function: null } },
      error,
      isLoading: false,
      isSkipped: false,
      refetch: vi.fn(),
    });

    const result = useRunCount();

    expect(result.kind).toBe('count-error');
    expect(result.error).toBe(error);
    expect(result.data).toBeUndefined();
  });

  it('reports a missing function when the query itself succeeded', () => {
    mocks.useSkippableGraphQLQuery.mockReturnValue({
      data: { environment: { function: null } },
      error: undefined,
      isLoading: false,
      isSkipped: false,
      refetch: vi.fn(),
    });

    const result = useRunCount();

    expect(result.kind).toBe('function-not-found');
    expect(result.error?.message).toBe('function not found');
  });

  it('returns the cancellation run count after a successful query', () => {
    mocks.useSkippableGraphQLQuery.mockReturnValue({
      data: { environment: { function: { cancellationRunCount: 42 } } },
      error: undefined,
      isLoading: false,
      isSkipped: false,
      refetch: vi.fn(),
    });

    const result = useRunCount();

    expect(result.kind).toBe('success');
    expect(result.data).toBe(42);
    expect(result.error).toBeUndefined();
  });
});
