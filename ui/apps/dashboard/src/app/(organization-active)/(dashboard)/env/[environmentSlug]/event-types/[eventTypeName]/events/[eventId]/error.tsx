'use client';

import { useEffect } from 'react';
import * as Sentry from '@sentry/nextjs';

import { FatalError } from '@/components/FatalError';

type EventErrorPops = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function EventError({ error, reset }: EventErrorPops) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return <FatalError error={error} reset={reset} />;
}
