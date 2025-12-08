"use client";

import { useEffect, useState } from "react";
import { usePathname, useSearchParams } from "next/navigation";
import { useOrganization, useUser } from "@clerk/nextjs";

import { analytics } from "@/utils/segment";

export default function Analytics() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [lastUrl, setLastUrl] = useState<string>();
  const { user, isSignedIn } = useUser();
  const { organization } = useOrganization();

  useEffect(() => {
    // Avoid duplicate page view tracking by tracking the last URL
    const url = `${pathname}?${searchParams.toString()}`;
    if (url === lastUrl) {
      return;
    }
    setLastUrl(url);
    analytics.page(null, {
      ref: searchParams.get("ref"),
    });
  }, [pathname, searchParams, lastUrl]);

  useEffect(() => {
    if (!isSignedIn || !organization) return;
    // See tracking plan for traits
    analytics.identify(user.externalId, {
      email: user.primaryEmailAddress?.emailAddress,
      name: user.fullName,
      clerk_user_id: user.id,
    });
    if (organization.publicMetadata.accountID) {
      // Other properties are set on the server, so we don't need to set them all here
      analytics.group(organization.publicMetadata.accountID, {
        name: organization.name,
      });
    }
  }, [isSignedIn, organization, user]);

  return null;
}
