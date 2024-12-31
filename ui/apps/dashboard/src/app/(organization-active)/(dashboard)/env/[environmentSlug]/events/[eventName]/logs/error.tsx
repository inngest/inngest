'use client';

import { useEffect } from 'react';
import * as Sentry from '@sentry/nextjs';

import { FatalError } from '@/components/FatalError';

type EventLogsErrorPops = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function EventLogsError({ error, reset }: EventLogsErrorPops) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return <FatalError error={error} reset={reset} />;
}
