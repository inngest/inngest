'use client';

import { FatalError } from '@/components/FatalError';

export default function Page({ error, reset }: React.ComponentProps<typeof FatalError>) {
  return <FatalError error={error} reset={reset} />;
}
