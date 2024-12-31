'use client';

import { useEffect } from 'react';
import { Error as ErrorElement } from '@inngest/components/Error/Error';
import * as Sentry from '@sentry/nextjs';

type OrganizationSetupErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function OrganizationSetupError({ error }: OrganizationSetupErrorProps) {
  useEffect(() => {
    Sentry.captureException(new Error('Failed to set up organization', { cause: error }));
  }, [error]);

  return <ErrorElement message="Failed to set up your organization" />;
}
