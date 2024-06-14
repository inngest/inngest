import { useCallback } from 'react';
import { useClient } from 'urql';

import { graphql } from '@/gql';
import { useEnvironment } from '../../../environment-context';

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
        { requestPolicy: 'network-only' }
      );
      if (res.error) {
        throw res.error;
      }
      if (!res.data) {
        throw new Error('no data');
      }
      return res.data.env.appCheck;
    },
    [client, env.id]
  );
}
