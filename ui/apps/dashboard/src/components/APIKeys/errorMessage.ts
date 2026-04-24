import type { CombinedError } from 'urql';

// Extract a user-friendly message from a urql CombinedError. The default
// `err.message` is prefixed with "[GraphQL]"/"[Network]" and may concatenate
// multiple errors — we surface the first GraphQL message, or a generic
// network fallback, so we don't leak transport details to the UI.
export function apiKeyErrorMessage(
  err: CombinedError,
  fallback = 'Something went wrong. Please try again.',
): string {
  const gqlMsg = err.graphQLErrors?.[0]?.message?.trim();
  if (gqlMsg) return gqlMsg;
  if (err.networkError) return 'Network error. Please check your connection.';
  return fallback;
}
