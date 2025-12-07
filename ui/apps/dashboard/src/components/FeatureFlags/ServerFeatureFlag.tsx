import { auth, clerkClient } from "@clerk/tanstack-react-start/server";
import * as Sentry from "@sentry/tanstackstart-react";

import { getLaunchDarklyClient } from "./ServerLaunchDarkly";

export async function getBooleanFlag(
  flag: string,
  { defaultValue = false }: { defaultValue?: boolean } = {},
): Promise<boolean> {
  const { userId, orgId } = await auth();

  if (!userId) {
    throw new Error("ServerLaunchdarkly init failed: user is not logged in");
  }

  const clerk = await clerkClient();
  const user = await clerk.users.getUser(userId);

  if (!user) {
    throw new Error("ServerLaunchdarkly init failed: user is not logged in");
  }

  let organization:
    | Awaited<ReturnType<typeof clerk.organizations.getOrganization>>
    | undefined;

  if (orgId) {
    organization = await clerk.organizations.getOrganization({
      organizationId: orgId,
    });
  }

  try {
    const client = await getLaunchDarklyClient();

    const accountID =
      organization?.publicMetadata?.accountID &&
      typeof organization.publicMetadata.accountID === "string"
        ? organization.publicMetadata.accountID
        : "Unknown";

    const context = {
      account: {
        key: accountID,
        name: organization?.name ?? "Unknown",
      },
      kind: "multi",
      user: {
        anonymous: false,
        key: user?.externalId ?? "Unknown",
        name:
          `${user?.firstName ?? ""} ${user?.lastName ?? ""}`.trim() ||
          "Unknown",
      },
    } as const;

    const variation = await client.variation(flag, context, defaultValue);
    return variation;
  } catch (err) {
    Sentry.captureException(err);
    console.error("Failed to get LaunchDarkly variation", err);
    return false;
  }
}
