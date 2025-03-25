'use client';

import { useEffect } from 'react';
import * as Sentry from '@sentry/nextjs';

import { FatalError } from '@/components/FatalError';

type VercelIntegrationErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function VercelIntegrationError({ error, reset }: VercelIntegrationErrorProps) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return <FatalError error={error} reset={reset} />;
}
