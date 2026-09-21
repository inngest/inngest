import { createContext, useEffect, useState } from 'react';

import { useLDClient, withLDProvider } from 'launchdarkly-react-client-sdk';
import { useOrganization, useUser } from '@clerk/tanstack-react-start';

export const IdentificationContext = createContext({
  isIdentified: false,
  isSettled: false,
});

function LaunchDarkly({ children }: { children: React.ReactNode }) {
  const [isIdentified, setIsIdentified] = useState(false);
  const client = useLDClient();

  const { user, isLoaded: userLoaded } = useUser();
  const { organization, isLoaded: orgLoaded } = useOrganization();

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

  // JWT-authed accounts lack the Clerk IDs identify needs, so they settle
  // without ever identifying.
  const isSettled =
    isIdentified || (userLoaded && orgLoaded && (!accountID || !externalID));

  return (
    <IdentificationContext.Provider value={{ isIdentified, isSettled }}>
      {children}
    </IdentificationContext.Provider>
  );
}

let clientSideID: string;
if (import.meta.env.VITE_LAUNCH_DARKLY_CLIENT_ID) {
  clientSideID = import.meta.env.VITE_LAUNCH_DARKLY_CLIENT_ID;
} else {
  console.error('missing VITE_LAUNCH_DARKLY_CLIENT_ID');
  clientSideID = 'missing';
}

export const ClientFeatureFlagProvider = withLDProvider<any>({
  clientSideID,
  reactOptions: {
    useCamelCaseFlagKeys: false,
  },
})(LaunchDarkly);
