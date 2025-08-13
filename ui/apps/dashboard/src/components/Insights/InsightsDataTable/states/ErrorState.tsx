'use client';

import { Banner } from '@inngest/components/Banner/Banner';
import { Button } from '@inngest/components/Button/Button';

import { useInsightsStateMachineContext } from '../../InsightsStateMachineContext/InsightsStateMachineContext';

const FALLBACK_ERROR = 'Something went wrong. Please try again.';

export function ErrorState() {
  const { error, runQuery } = useInsightsStateMachineContext();

  return (
    <Banner
      cta={
        <Button
          appearance="ghost"
          kind="danger"
          label="Retry"
          onClick={() => {
            runQuery();
          }}
        />
      }
      severity="error"
    >
      {error ?? FALLBACK_ERROR}
    </Banner>
  );
}
