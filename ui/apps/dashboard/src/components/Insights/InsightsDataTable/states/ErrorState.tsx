'use client';

import { Alert } from '@inngest/components/Alert';

import { useInsightsQueryContext } from '../../context';

const FALLBACK_ERROR = 'Something went wrong. Please try again.';

export function ErrorState() {
  const { error, runQuery } = useInsightsQueryContext();

  return (
    <Alert className="text-sm" severity="error">
      {error ?? FALLBACK_ERROR}
    </Alert>
  );
}
