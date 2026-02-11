import { useMemo } from 'react';
import { createConnectTransport } from '@connectrpc/connect-web';
import { TransportProvider } from '@connectrpc/connect-query';
import type { ReactNode } from 'react';

//
// Use same base URL pattern as devApi.ts - env var or relative path
const getBaseUrl = () => {
  if (import.meta.env.VITE_PUBLIC_API_BASE_URL) {
    return new URL('/', import.meta.env.VITE_PUBLIC_API_BASE_URL)
      .toString()
      .replace(/\/$/, '');
  }
  return typeof window !== 'undefined' ? window.location.origin : '';
};

type ConnectRpcProviderProps = {
  children: ReactNode;
};

export const ConnectRpcProvider = ({ children }: ConnectRpcProviderProps) => {
  const transport = useMemo(
    () => createConnectTransport({ baseUrl: getBaseUrl() }),
    [],
  );

  return (
    <TransportProvider transport={transport}>{children}</TransportProvider>
  );
};
