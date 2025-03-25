'use client';

import { createContext, useEffect, useState } from 'react';
import { useOrganization, useUser } from '@clerk/nextjs';
import { useLDClient, withLDProvider } from 'launchdarkly-react-client-sdk';

export const IdentificationContext = createContext({ isIdentified: false });

function LaunchDarkly({ children }: { children: React.ReactNode }) {
  const [isIdentified, setIsIdentified] = useState(false);
  const client = useLDClient();

  const { user } = useUser();
  const { organization } = useOrganization();

  const accountID = organization?.publicMetadata.accountID;
  const externalID = user?.externalId;
  const userName = user?.fullName;

  useEffect(() => {
    if (!client || !accountID || !externalID) {
      return;
    }

    client
      .identify({
        kind: 'multi',
        account: {
          key: accountID,
          name: organization.name,
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
  }, [accountID, client, externalID, organization?.name, userName]);

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
