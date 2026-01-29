import { useMemo } from 'react';
import { useAuth } from '@clerk/tanstack-react-start';
import { TransportProvider } from '@connectrpc/connect-query';
import { createConnectTransport } from '@connectrpc/connect-web';

type Props = {
  children: React.ReactNode;
};

//
// ConnectRpcProvider wraps children with the connect-query TransportProvider,
// configured with auth interceptor for Clerk tokens.
export const ConnectRpcProvider = ({ children }: Props) => {
  const { getToken } = useAuth();

  const transport = useMemo(
    () =>
      createConnectTransport({
        baseUrl: import.meta.env.VITE_API_URL,
        useBinaryFormat: true,
        interceptors: [
          (next) => async (req) => {
            const token = await getToken();
            if (token) {
              req.header.set('Authorization', `Bearer ${token}`);
            }
            return next(req);
          },
        ],
        fetch: (input, init) =>
          fetch(input, { ...init, credentials: 'include' }),
      }),
    [getToken],
  );

  return (
    <TransportProvider transport={transport}>{children}</TransportProvider>
  );
};
