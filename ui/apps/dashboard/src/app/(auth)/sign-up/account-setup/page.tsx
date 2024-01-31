'use client';

import { useEffect, useState } from 'react';
import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { useAuth, useOrganization, useOrganizationList, useUser } from '@clerk/nextjs';
import { ExclamationCircleIcon } from '@heroicons/react/20/solid';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import LoadingIcon from '@/icons/LoadingIcon';

export default function AccountSetupPage() {
  const router = useRouter();
  const secondsElapsed = useSecondsElapsed();
  const { getToken } = useAuth();
  const { user } = useUser();
  useForceActiveOrganization();
  const isAccountSetup = useIsAccountSetup();

  // Reload memberships if the user doesn't have an active organization
  useEffect(() => {
    if (!isLoaded || userMemberships.data[0] || organization) return;

    const intervalID = setInterval(() => {
      userMemberships.revalidate();
    }, 500);

    return () => {
      clearInterval(intervalID);
    };
  }, [isLoaded, organization, setActive, userMemberships, userMemberships.data]);

  // Reload the organization while it isn't yet set up
  useEffect(() => {
    if (!organization || organization.publicMetadata.accountID) return;

    const intervalID = setInterval(() => {
      organization.reload();
    }, 500);

    return () => {
      clearInterval(intervalID);
    };
  }, [isAccountSetup, organization]);

  // Redirect to the home page once the account is set up
  useEffect(() => {
    if (!isAccountSetup) return;

    // We need to refresh the token before redirecting so that the token contains the account ID
    getToken({ skipCache: true }).then(() => {
      // We use `replace` so that the user doesn't get redirected back to this page if they click the back button
      router.replace(process.env.NEXT_PUBLIC_HOME_PATH as Route);
    });
  }, [getToken, isAccountSetup, router]);

  useEffect(() => {
    if (secondsElapsed !== 10) return;

    window.inngest.send({
      name: 'app/account.setup.delayed',
      data: {
        userID: user?.id,
        delayInSeconds: secondsElapsed,
      },
      user,
      v: '2023-08-31.1',
    });
  }, [secondsElapsed, user]);

  if (secondsElapsed > 30) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center gap-5">
        <div className="inline-flex items-center gap-2 text-red-600">
          <ExclamationCircleIcon className="h-4 w-4" />
          <h2 className="text-sm">Failed to set up your account</h2>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full w-full items-center justify-center">
      <LoadingIcon />
    </div>
  );
}

/**
 * This forces the user to have an active organization when the organizations feature is not
 * enabled. When the organizations feature is enabled, the user will already have an active
 * organization set up by the `/organization-list` page.
 */
function useForceActiveOrganization() {
  const { value: isOrganizationsEnabled } = useBooleanFlag('organizations');
  const { isLoaded, userMemberships } = useOrganizationList({
    userMemberships: {
      infinite: true,
    },
  });

  const { getToken } = useAuth();

  const { isLoaded: isOrganizationLoaded, organization } = useOrganization();
  const router = useRouter();

  const newOrganizationID = userMemberships.data?.[0]?.organization.id;

  useEffect(() => {
    if (
      !isOrganizationsEnabled &&
      isLoaded &&
      isOrganizationLoaded &&
      newOrganizationID &&
      !organization
    ) {
      getToken({ skipCache: true }).then(() => {
        router.replace('/organization-list' as Route);
      });
    }
  }, [
    isLoaded,
    isOrganizationLoaded,
    isOrganizationsEnabled,
    organization,
    newOrganizationID,
    router,
    getToken,
  ]);

  // Reload memberships if the user doesn't have an active organization
  useEffect(() => {
    if (isOrganizationsEnabled || !isLoaded || newOrganizationID || organization) return;

    const intervalID = setInterval(() => {
      userMemberships.revalidate();
    }, 500);

    return () => {
      clearInterval(intervalID);
    };
  }, [isLoaded, isOrganizationsEnabled, organization, userMemberships, newOrganizationID]);
}

/**
 * This hook returns true if the user's account is set up. It will reload the organization until
 * the account is set up.
 *
 * @returns {boolean}
 */
function useIsAccountSetup(): boolean {
  const { isLoaded, organization } = useOrganization();

  const isAccountSetup = Boolean(organization?.publicMetadata.accountID);

  // Reload the organization until the account is set up
  useEffect(() => {
    if (!isLoaded || !organization || isAccountSetup) return;

    const intervalID = setInterval(() => {
      organization.reload();
    }, 500);

    return () => {
      clearInterval(intervalID);
    };
  }, [isAccountSetup, isLoaded, organization]);

  return isAccountSetup;
}

function useSecondsElapsed() {
  const [seconds, setSeconds] = useState(0);
  useEffect(() => {
    const intervalID = setInterval(() => {
      setSeconds((seconds) => seconds + 1);
    }, 1_000);

    return () => {
      clearInterval(intervalID);
    };
  }, []);

  return seconds;
}
