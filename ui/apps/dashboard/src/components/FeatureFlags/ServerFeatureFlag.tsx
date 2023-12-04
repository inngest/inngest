// import type { PropsWithChildren } from 'react';
// import { currentUser } from '@clerk/nextjs';
// import * as Sentry from '@sentry/nextjs';
//
// import { getLaunchDarklyClient } from '@/launchDarkly';
//
// type Props = PropsWithChildren<{
//   defaultValue?: boolean;
//   flag: string;
// }>;
//
// // Conditionally renders children based on a feature flag.
// export async function ServerFeatureFlag({ children, defaultValue = false, flag }: Props) {
//   const isEnabled = await getBooleanFlag(flag, { defaultValue });
//   if (isEnabled) {
//     return <>{children}</>;
//   }
//
//   return null;
// }
//
// export async function getBooleanFlag(
//   flag: string,
//   { defaultValue = false }: { defaultValue?: boolean } = {}
// ): Promise<boolean> {
//   const user = await currentUser();
//
//   try {
//     const client = await getLaunchDarklyClient();
//
//     const accountID =
//       user?.publicMetadata.accountID && typeof user?.publicMetadata.accountID === 'string'
//         ? user?.publicMetadata.accountID
//         : 'Unknown';
//
//     const context = {
//       account: {
//         key: accountID,
//         name: 'Unknown', // TODO: Add account name whenever we have adopted Clerk Organizations
//       },
//       kind: 'multi',
//       user: {
//         anonymous: false,
//         key: user?.externalId ?? 'Unknown',
//         name: `${user?.firstName ?? ''} ${user?.lastName ?? ''}`.trim() || 'Unknown',
//       },
//     } as const;
//
//     const variation = await client.variation(flag, context, defaultValue);
//     return variation;
//   } catch (err) {
//     Sentry.captureException(err);
//     console.error('Failed to get LaunchDarkly variation', err);
//     return false;
//   }
// }
