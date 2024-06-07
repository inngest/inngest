'use client';

import { useEffect } from 'react';
import { Button } from '@inngest/components/Button';
import { RiErrorWarningLine, RiLoopLeftLine } from '@remixicon/react';
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
    <div className="flex h-full w-full flex-col items-center justify-center gap-5">
      <div className="inline-flex items-center gap-2 text-red-600">
        <RiErrorWarningLine className="h-4 w-4" />
        <h2 className="text-sm">Failed to load logs for event</h2>
      </div>
      <Button
        appearance="outlined"
        iconSide="right"
        icon={<RiLoopLeftLine className=" text-slate-700" />}
        onClick={() => reset()}
        label="Reload"
      />
    </div>
  );
}
