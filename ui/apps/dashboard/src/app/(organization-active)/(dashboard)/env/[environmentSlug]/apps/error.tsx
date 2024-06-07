'use client';

import { useEffect } from 'react';
import { Button } from '@inngest/components/Button';

type Props = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function Page({ error, reset }: Props) {
  useEffect(() => {
    // Log the error to an error reporting service
    console.error(error);
  }, [error]);

  return (
    <div className="flex h-screen items-center justify-center">
      <div className="w-[600px] rounded border border-slate-300 p-4">
        <h2>Something went wrong!</h2>

        <div className="my-6 overflow-scroll rounded bg-slate-200 p-2">{error.message}</div>

        <Button
          onClick={
            // Attempt to recover by trying to re-render the segment
            () => reset()
          }
          label="Try again"
        />
      </div>
    </div>
  );
}
