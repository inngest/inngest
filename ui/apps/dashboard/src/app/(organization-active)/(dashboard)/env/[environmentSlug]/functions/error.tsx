'use client';

import { useEffect } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import * as Sentry from '@sentry/nextjs';

type FunctionListErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function FunctionListError({ error, reset }: FunctionListErrorProps) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-5">
      <Alert className="mb-4" severity="error">
        <p className="mb-4">Something went wrong!</p>

        <pre className="w-full overflow-scroll rounded-md border border-slate-300 bg-slate-100 p-1 text-slate-800 ">
          {error.message}
        </pre>
      </Alert>
      <Button onClick={() => reset()} kind="primary" label="Try Again" />
    </div>
  );
}
