import { useAuth } from '@clerk/tanstack-react-start';
import { useQuery } from '@tanstack/react-query';

import { oauthRequest } from './oauthRequest';

export type OAuthSession = {
  id: string;
  name: string;
  client_name: string;
  environment: { id: string; name: string } | null;
  permissions: string[];
  created_at: string;
  expires_at: string;
  revoked_at: string | null;
};

export type OAuthSessionPage = {
  sessions: OAuthSession[];
  has_more: boolean;
};

export function useOAuthSessions(offset: number) {
  const { getToken, userId, orgId } = useAuth();
  return useQuery({
    queryKey: ['oauth-sessions', userId, orgId, offset],
    queryFn: ({ signal }) =>
      oauthRequest<OAuthSessionPage>(
        getToken,
        `/oauth/sessions?limit=20&offset=${offset}`,
        undefined,
        signal,
      ),
    staleTime: 0,
    retry: false,
  });
}
