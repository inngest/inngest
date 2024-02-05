'use client';

import { useEffect } from 'react';
import { useOrganization, useUser } from '@clerk/nextjs';
import * as Sentry from '@sentry/nextjs';

export default function SentryUserIdentification() {
  const { user, isSignedIn } = useUser();
  const { organization } = useOrganization();

  useEffect(() => {
    if (!isSignedIn) return;

    const baseUser = {
      ...(user.externalId && { id: user.externalId }),
      clerk_user_id: user.id,
      email: user.primaryEmailAddress?.emailAddress,
    };

    const accountID = organization?.publicMetadata.accountID;

    if (typeof accountID !== 'undefined' && typeof accountID !== 'string') {
      Sentry.setUser(baseUser);
      Sentry.captureException(
        new Error('Expected organization.publicMetadata.accountID to be a string when defined.')
      );
      return;
    }

    Sentry.setUser({
      ...baseUser,
      ...(accountID && { account_id: accountID }),
    });
  }, [
    isSignedIn,
    user?.id,
    user?.externalId,
    user?.primaryEmailAddress?.emailAddress,
    organization?.publicMetadata.accountID,
  ]);

  return null;
}
