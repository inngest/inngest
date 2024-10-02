import type { PropsWithChildren } from 'react';
import { auth, clerkClient, currentUser } from '@clerk/nextjs/server';
import * as Sentry from '@sentry/nextjs';

import { getLaunchDarklyClient } from '@/launchDarkly';

type Props = PropsWithChildren<{
  defaultValue?: boolean;
  flag: string;
}>;

// Conditionally renders children based on a feature flag.
export async function ServerFeatureFlag({ children, defaultValue = false, flag }: Props) {
  const isEnabled = await getBooleanFlag(flag, { defaultValue });
  if (isEnabled) {
    return <>{children}</>;
  }

  return null;
}

export async function getBooleanFlag(
  flag: string,
  { defaultValue = false }: { defaultValue?: boolean } = {}
): Promise<boolean> {
  const user = await currentUser();
  const clerk = clerkClient();
  const { orgId } = auth();

  let organization: Awaited<ReturnType<typeof clerk.organizations.getOrganization>> | undefined;

  if (orgId) {
    organization = await clerk.organizations.getOrganization({ organizationId: orgId });
  }

  try {
    const client = await getLaunchDarklyClient();

    const accountID =
      organization?.publicMetadata?.accountID &&
      typeof organization.publicMetadata.accountID === 'string'
        ? organization.publicMetadata.accountID
        : 'Unknown';

    const context = {
      account: {
        key: accountID,
        name: organization?.name ?? 'Unknown',
      },
      kind: 'multi',
      user: {
        anonymous: false,
        key: user?.externalId ?? 'Unknown',
        name: `${user?.firstName ?? ''} ${user?.lastName ?? ''}`.trim() || 'Unknown',
      },
    } as const;

    const variation = await client.variation(flag, context, defaultValue);
    return variation;
  } catch (err) {
    Sentry.captureException(err);
    console.error('Failed to get LaunchDarkly variation', err);
    return false;
  }
}
