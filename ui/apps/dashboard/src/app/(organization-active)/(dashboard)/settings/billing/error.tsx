'use client';

import { useEffect } from 'react';
import { ArrowPathIcon, ExclamationCircleIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import * as Sentry from '@sentry/nextjs';

type BillingErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function BillingError({ error, reset }: BillingErrorProps) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-5 bg-slate-100">
      <div className="max-w-4xl rounded-md border border-slate-200 bg-white p-4 text-slate-500">
        <div className="inline-flex items-center gap-2 text-red-600">
          <ExclamationCircleIcon className="h-4 w-4" />
          Failed To Load Billing Page
        </div>
        <div>
          <details className="pt-4">
            <summary className="text-sm">Error</summary>
            <div className="mt-4 font-mono text-xs">{error.message}</div>
          </details>
        </div>
      </div>
      <Button
        appearance="outlined"
        iconSide="right"
        icon={<ArrowPathIcon className=" text-slate-700" />}
        btnAction={() => reset()}
        label="Reload"
      />
    </div>
  );
}
