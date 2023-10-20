'use client';

import { createContext, useEffect, useState } from 'react';
import { useUser } from '@clerk/nextjs';
import { useLDClient, withLDProvider } from 'launchdarkly-react-client-sdk';
import { useQuery } from 'urql';

import { graphql } from '@/gql';

const GetAccountNameDocument = graphql(`
  query GetAccountName {
    account {
      name
    }
  }
`);

export const IdentificationContext = createContext({ isIdentified: false });

function LaunchDarkly({ children }: { children: React.ReactNode }) {
  const [isIdentified, setIsIdentified] = useState(false);
  const client = useLDClient();

  const { user } = useUser();

  const accountID = user?.publicMetadata.accountID;
  const externalID = user?.externalId;
  const userName = user?.fullName;

  const [{ data }] = useQuery({ query: GetAccountNameDocument, pause: !accountID });
  const accountName = data?.account?.name;

  useEffect(() => {
    if (!client || !accountID || !externalID) {
      return;
    }

    console.debug('Identify user', { accountID, externalID, userName });

    client
      .identify({
        kind: 'multi',
        account: {
          key: accountID,
          name: accountName ?? 'Unknown', // TODO: replace this with organization name whenever we have adopted Clerk Organizations
        },
        user: {
          anonymous: false,
          key: externalID,
          name: userName ?? 'Unknown',
        },
      })
      .then(() => {
        // We need to set this because calling client.identify won't trigger a
        // re-render for the useLDClient hook.
        setIsIdentified(true);
      });
  }, [accountID, accountName, client, externalID, userName]);

  return (
    <IdentificationContext.Provider value={{ isIdentified }}>
      {children}
    </IdentificationContext.Provider>
  );
}

let clientSideID: string;
if (process.env.NEXT_PUBLIC_LAUNCH_DARKLY_CLIENT_ID) {
  clientSideID = process.env.NEXT_PUBLIC_LAUNCH_DARKLY_CLIENT_ID;
} else {
  console.error('missing NEXT_PUBLIC_LAUNCH_DARKLY_CLIENT_ID');
  clientSideID = 'missing';
}

export const ClientFeatureFlagProvider = withLDProvider<any>({
  clientSideID,
  reactOptions: {
    useCamelCaseFlagKeys: false,
  },
})(LaunchDarkly);
