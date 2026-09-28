import type { CombinedError } from 'urql';

export function secretErrorMessage(
  error: CombinedError,
  fallback: string,
): string {
  const message = error.graphQLErrors[0]?.message;
  if (message === 'forbidden')
    return 'Only organization admins can manage secrets.';
  if (message === 'sandbox secrets are not configured') {
    return 'Secrets are not available in this environment yet.';
  }
  // Do not display arbitrary mutation/provider errors: they can contain input.
  return fallback;
}
