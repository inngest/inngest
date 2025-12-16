import type { PropsWithChildren } from 'react';
import { auth } from '@clerk/nextjs/server';
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
  const { sessionClaims } = auth();

  try {
    const client = await getLaunchDarklyClient();

    const accountID = sessionClaims?.orgPublickMetadata?.accountId ?? 'Unknown';

    const context = {
      account: {
        key: accountID,
        name: sessionClaims?.orgName ?? 'Unknown',
      },
      kind: 'multi',
      user: {
        anonymous: false,
        key: sessionClaims?.externalId ?? 'Unknown',
        name: sessionClaims?.fullName || 'Unknown',
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
