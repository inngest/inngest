import { useOrganization, useUser } from '@clerk/tanstack-react-start';
import { useLocation, useSearch } from '@tanstack/react-router';
import { useEffect, useState } from 'react';

import { analytics } from '@/utils/segment';

export default function SegmentAnalytics() {
  const location = useLocation();
  const search = useSearch({ strict: false });
  const [lastUrl, setLastUrl] = useState<string>();
  const { user, isSignedIn } = useUser();
  const { organization } = useOrganization();

  useEffect(() => {
    // Avoid duplicate page view tracking by tracking the last URL
    if (location.href === lastUrl) {
      return;
    }
    setLastUrl(location.href);

    analytics.page(null, {
      ref: 'ref' in search ? search.ref : null,
    });
  }, [location, search, lastUrl]);

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
