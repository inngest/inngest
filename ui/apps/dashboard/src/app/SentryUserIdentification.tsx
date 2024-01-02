'use client';

import { useEffect } from 'react';
import { useUser } from '@clerk/nextjs';
import * as Sentry from '@sentry/nextjs';

export default function SentryUserIdentification() {
  const { user, isSignedIn } = useUser();

  useEffect(() => {
    if (!isSignedIn) return;

    Sentry.setUser({
      ...(user.externalId && { id: user.externalId }),
      clerk_user_id: user.id,
      email: user.primaryEmailAddress?.emailAddress,
    });
  }, [isSignedIn, user?.externalId, user?.id, user?.primaryEmailAddress?.emailAddress]);

  return null;
}
