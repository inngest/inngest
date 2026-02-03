import { useMemo } from 'react';
import { useAuth } from '@clerk/tanstack-react-start';
import { TransportProvider } from '@connectrpc/connect-query';
import { createConnectTransport } from '@connectrpc/connect-web';
import type { Interceptor } from '@connectrpc/connect';

type Props = {
  children: React.ReactNode;
};

// ConnectRpcProvider wraps children with the connect-query TransportProvider,
// configured with an auth interceptor for Clerk tokens.
export const ConnectRpcProvider = ({ children }: Props) => {
  const { getToken } = useAuth();

  const authInterceptor: Interceptor = useMemo(
    () => (next) => async (req) => {
      const token = await getToken();
      if (token) {
        req.header.set('Authorization', `Bearer ${token}`);
      }
      return next(req);
    },
    [getToken],
  );

  //
  // Connect transport for unary + streaming calls
  const connectTransport = useMemo(
    () =>
      createConnectTransport({
        baseUrl: import.meta.env.VITE_API_URL,
        useBinaryFormat: true,
        interceptors: [authInterceptor],
        fetch: (input, init) =>
          fetch(input, { ...init, credentials: 'include' }),
      }),
    [authInterceptor],
  );

  return (
    <TransportProvider transport={connectTransport}>
      {children}
    </TransportProvider>
  );
};
