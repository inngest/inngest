'use client';

import { useEffect } from 'react';
import { Button } from '@inngest/components/Button';
import { Error as ErrorElement } from '@inngest/components/Error/Error';
import { RiLoopLeftLine } from '@remixicon/react';
import * as Sentry from '@sentry/nextjs';

type EventLogsErrorPops = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function EventLogsError({ error, reset }: EventLogsErrorPops) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <ErrorElement
      message="Failed to load logs for event"
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
