'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Provider } from 'react-redux';

import { store } from '@/store/store';
import { SharedDataProvider } from './SharedDataProvider';

type StoreProviderProps = {
  children: React.ReactNode;
};

export const queryClient = new QueryClient();

export default function StoreProvider({ children }: StoreProviderProps) {
  return (
    <QueryClientProvider client={queryClient}>
      <Provider store={store}>
        <SharedDataProvider>{children}</SharedDataProvider>
      </Provider>
    </QueryClientProvider>
  );
}
