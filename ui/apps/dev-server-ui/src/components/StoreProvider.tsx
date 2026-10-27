import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Provider } from 'react-redux';

import { QuickstartProvider } from './Quickstart/QuickstartProvider';
import { store } from '@/store/store';
import { SharedContextProvider } from './SharedContextProvider';

type StoreProviderProps = {
  children: React.ReactNode;
};

export const queryClient = new QueryClient();

export default function StoreProvider({ children }: StoreProviderProps) {
  return (
    <QueryClientProvider client={queryClient}>
      <Provider store={store}>
        <SharedContextProvider>
          <QuickstartProvider>{children}</QuickstartProvider>
        </SharedContextProvider>
      </Provider>
    </QueryClientProvider>
  );
}
