import { CombinedError } from 'urql';
import { describe, expect, it } from 'vitest';

import { apiKeyErrorMessage } from '@/components/APIKeys/errorMessage';

describe('apiKeyErrorMessage', () => {
  it('returns the first GraphQL error message, stripping the urql prefix', () => {
    const err = new CombinedError({
      graphQLErrors: [{ message: 'Name already in use' } as Error],
    });
    expect(apiKeyErrorMessage(err)).toBe('Name already in use');
  });

  it('picks the first GraphQL error when multiple are returned', () => {
    const err = new CombinedError({
      graphQLErrors: [
        { message: 'first' } as Error,
        { message: 'second' } as Error,
      ],
    });
    expect(apiKeyErrorMessage(err)).toBe('first');
  });

  it('trims whitespace from the GraphQL message', () => {
    const err = new CombinedError({
      graphQLErrors: [{ message: '  padded  ' } as Error],
    });
    expect(apiKeyErrorMessage(err)).toBe('padded');
  });

  it('falls back to a friendly network message when only a network error is present', () => {
    const err = new CombinedError({
      networkError: new Error('connection refused'),
    });
    expect(apiKeyErrorMessage(err)).toBe(
      'Network error. Please check your connection.',
    );
  });

  it('prefers a GraphQL message over a network error when both are present', () => {
    const err = new CombinedError({
      graphQLErrors: [{ message: 'forbidden' } as Error],
      networkError: new Error('boom'),
    });
    expect(apiKeyErrorMessage(err)).toBe('forbidden');
  });

  it('returns the provided fallback when the GraphQL message is blank', () => {
    const err = new CombinedError({
      graphQLErrors: [{ message: '   ' } as Error],
    });
    expect(apiKeyErrorMessage(err, 'Could not do the thing.')).toBe(
      'Could not do the thing.',
    );
  });

  it('returns the default fallback when no error details are available', () => {
    const err = new CombinedError({ graphQLErrors: [] });
    expect(apiKeyErrorMessage(err)).toBe(
      'Something went wrong. Please try again.',
    );
  });
});
