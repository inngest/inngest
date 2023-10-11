'use client';

import { Alert } from '@/components/Alert';
import Button from '@/components/Button';

export default function ErrorPage({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-5">
      <Alert className="mb-4" severity="error">
        <p className="mb-4">Something went wrong!</p>

        <pre className="w-full overflow-scroll rounded-md border border-slate-300 bg-slate-100 p-1 text-slate-800 ">
          {error.message}
        </pre>
      </Alert>
      <Button onClick={() => reset()}>Try again</Button>
    </div>
  );
}
