import { createContext, useEffect, useState } from 'react';

import {
  useLDClient,
  useLDClientError,
  withLDProvider,
} from 'launchdarkly-react-client-sdk';
import { useOrganization, useUser } from '@clerk/tanstack-react-start';

export const IdentificationContext = createContext({
  isIdentified: false,
  hasError: false,
});

function LaunchDarkly({ children }: { children: React.ReactNode }) {
  const [identification, setIdentification] = useState<{
    accountID: unknown;
    externalID: string | null | undefined;
    status: 'ready' | 'error';
  }>();
  const client = useLDClient();
  const clientError = useLDClientError();

  const { user } = useUser();
  const { organization } = useOrganization();

  const accountID = organization?.publicMetadata.accountID;
  const externalID = user?.externalId;
  const userName = user?.fullName;

  useEffect(() => {
    if (!client || !accountID || !externalID) {
      return;
    }

    let active = true;
    setIdentification(undefined);

    Promise.resolve()
      .then(() =>
        client.identify({
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
        }),
      )
      .then(() => {
        if (active)
          setIdentification({ accountID, externalID, status: 'ready' });
      })
      .catch((error: unknown) => {
        if (!active) return;
        console.error(
          "Couldn't identify feature flag context; using flag defaults",
          error,
        );
        setIdentification({ accountID, externalID, status: 'error' });
      });

    return () => {
      active = false;
    };
  }, [accountID, client, externalID, organization?.name, userName]);

  const status =
    identification?.accountID === accountID &&
    identification?.externalID === externalID
      ? identification?.status
      : undefined;
  const hasError = Boolean(clientError) || status === 'error';

  return (
    <IdentificationContext.Provider
      value={{ isIdentified: status === 'ready', hasError }}
    >
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
