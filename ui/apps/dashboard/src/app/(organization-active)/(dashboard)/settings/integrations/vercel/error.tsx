'use client';

import { useEffect } from 'react';
import { Button } from '@inngest/components/Button';
import { Error as ErrorElement } from '@inngest/components/Error/Error';
import { RiLoopLeftLine } from '@remixicon/react';
import * as Sentry from '@sentry/nextjs';

type VercelIntegrationErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function VercelIntegrationError({ error, reset }: VercelIntegrationErrorProps) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <ErrorElement
      message="Failed to load Vercel integration settings"
      button={
        <Button
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
