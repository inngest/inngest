import { useCallback } from 'react';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { fetchWithTimeout } from '@/queries/fetch';

const query = graphql(`
  query CheckApp($envID: ID!, $url: String!) {
    env: workspace(id: $envID) {
      appCheck(url: $url) {
        apiOrigin {
          value
        }
        appID {
          value
        }
        authenticationSucceeded {
          value
        }
        env {
          value
        }
        error
        eventAPIOrigin {
          value
        }
        eventKeyStatus
        extra
        framework {
          value
        }
        isReachable
        isSDK
        mode
        respHeaders
        respStatusCode
        sdkLanguage {
          value
        }
        sdkVersion {
          value
        }
        serveOrigin {
          value
        }
        servePath {
          value
        }
        signingKeyStatus
        signingKeyFallbackStatus
      }
    }
  }
`);

export function useGetAppInfo() {
  const client = useClient();
  const env = useEnvironment();

  return useCallback(
    async (url: string) => {
      const res = await client.query(
        query,
        {
          envID: env.id,
          url,
        },
        {
          fetch: fetchWithTimeout(5_000),
          requestPolicy: 'network-only',
        }
      );
      if (res.error) {
        if (res.error.message.includes('signal is aborted')) {
          throw new Error(`Request timed out`);
        }

        throw res.error;
      }
      if (!res.data) {
        throw new Error('No data');
      }
      return res.data.env.appCheck;
    },
    [client, env.id]
  );
}
