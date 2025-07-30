'use client';

import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button/Button';

import { useInsightsQueryContext } from '../../context';

const FALLBACK_ERROR = 'Something went wrong. Please try again.';

export function ErrorState() {
  const { error, runQuery } = useInsightsQueryContext();

  return (
    <Alert
      className="rounded-none text-sm"
      inlineButton={
        <Button
          appearance="solid"
          className="ml-auto h-auto p-0 text-sm font-medium underline"
          kind="secondary"
          label="Retry"
          size="medium"
          onClick={() => {
            runQuery();
          }}
        />
      }
      severity="error"
    >
      {error ?? FALLBACK_ERROR}
    </Alert>
  );
}
