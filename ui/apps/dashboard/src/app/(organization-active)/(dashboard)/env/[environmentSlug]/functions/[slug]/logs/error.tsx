'use client';

import { useEffect } from 'react';
import { NewButton } from '@inngest/components/Button';
import { Error as ErrorElement } from '@inngest/components/Error/Error';
import { RiLoopLeftLine } from '@remixicon/react';
import * as Sentry from '@sentry/nextjs';

type FunctionRunsErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function FunctionRunsError({ error, reset }: FunctionRunsErrorProps) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <ErrorElement
      message="Failed to load log"
      button={
        <NewButton
          label="Reload"
          appearance="outlined"
          iconSide="right"
          icon={<RiLoopLeftLine />}
          onClick={() => reset()}
        />
      }
    />
  );
}
