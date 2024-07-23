'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { CancellationTable } from './CancellationTable';

type Props = {
  params: {
    environmentSlug: string;
    slug: string;
  };
};

const queryClient = new QueryClient();

export default function Page({ params }: Props) {
  const envSlug = decodeURIComponent(params.environmentSlug);
  const fnSlug = decodeURIComponent(params.slug);

  return (
    <QueryClientProvider client={queryClient}>
      <CancellationTable envSlug={envSlug} fnSlug={fnSlug} />
    </QueryClientProvider>
  );
}
