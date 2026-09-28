import { createContext, useEffect, useState } from 'react';

import {
  useLDClient,
  useLDClientError,
  withLDProvider,
} from 'launchdarkly-react-client-sdk';
import { useOrganization, useUser } from '@clerk/tanstack-react-start';

export const IDENTIFICATION_TIMEOUT_MS = 5_000;

export const IdentificationContext = createContext({
  isIdentified: false,
  hasError: false,
  isUnavailable: false,
});

function LaunchDarkly({ children }: { children: React.ReactNode }) {
  const [identification, setIdentification] = useState<{
    accountID: unknown;
    externalID: string | null | undefined;
    status: 'ready' | 'error';
  }>();
  const client = useLDClient();
  const clientError = useLDClientError();

  const { isLoaded: isUserLoaded, user } = useUser();
  const { isLoaded: isOrganizationLoaded, organization } = useOrganization();

  const accountID = organization?.publicMetadata.accountID;
  const externalID = user?.externalId;
  const userName = user?.fullName;

  useEffect(() => {
    if (!client || !accountID || !externalID) {
      return;
    }

    let active = true;
    let settled = false;
    setIdentification(undefined);

    const timeout = window.setTimeout(() => {
      if (!active || settled) return;
      settled = true;
      console.error(
        "Couldn't identify feature flag context before timeout; using flag defaults",
      );
      setIdentification({ accountID, externalID, status: 'error' });
    }, IDENTIFICATION_TIMEOUT_MS);

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
        if (!active || settled) return;
        settled = true;
        window.clearTimeout(timeout);
        setIdentification({ accountID, externalID, status: 'ready' });
      })
      .catch((error: unknown) => {
        if (!active || settled) return;
        settled = true;
        window.clearTimeout(timeout);
        console.error(
          "Couldn't identify feature flag context; using flag defaults",
          error,
        );
        setIdentification({ accountID, externalID, status: 'error' });
      });

    return () => {
      active = false;
      window.clearTimeout(timeout);
    };
  }, [accountID, client, externalID, organization?.name, userName]);

  const status =
    identification?.accountID === accountID &&
    identification?.externalID === externalID
      ? identification?.status
      : undefined;
  const hasIdentity = Boolean(accountID && externalID);
  const hasError = Boolean(clientError) || status === 'error';
  const isUnavailable = isUserLoaded && isOrganizationLoaded && !hasIdentity;

  return (
    <IdentificationContext.Provider
      value={{
        isIdentified: status === 'ready',
        hasError,
        isUnavailable,
      }}
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
