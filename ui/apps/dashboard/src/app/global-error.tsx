'use client';

import { useEffect } from 'react';
import { RiErrorWarningLine } from '@remixicon/react';
import * as Sentry from '@sentry/nextjs';

type GlobalErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function GlobalError({ error }: GlobalErrorProps) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <html>
      <body>
        <div className="flex h-full w-full flex-col items-center justify-center gap-5">
          <div className="inline-flex items-center gap-2 text-yellow-600">
            <RiErrorWarningLine className="h-4 w-4" />
            <h2 className="text-sm">Something went wrong!</h2>
          </div>
        </div>
      </body>
    </html>
  );
}
