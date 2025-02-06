'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { URQLProvider } from '@/queries/URQLProvider';
import { Shared } from '../Shared';

const queryClient = new QueryClient();

export function ClientSideProviders({ children }: React.PropsWithChildren) {
  return (
    <QueryClientProvider client={queryClient}>
      <URQLProvider>{children}</URQLProvider>
    </QueryClientProvider>
  );
}
